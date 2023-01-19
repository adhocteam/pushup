package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
)

type testRequest struct {
	name           string
	path           string
	queryParams    []string
	expectedOutput string
}

func TestPushup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	tmpdir := t.TempDir()
	pushup := filepath.Join(tmpdir, "pushup.exe")

	// build Pushup executable
	if err := exec.Command("go", "build", "-o", pushup, ".").Run(); err != nil {
		t.Fatalf("building Pushup exe: %v", err)
	}

	testdataDir := "./testdata"
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("reading testdata dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), upFileExt) {
			t.Run(entry.Name(), func(t *testing.T) {
				// FIXME(paulsmith): remove this once we have been panic
				// handling in layouts
				if entry.Name() == "panicking.up" {
					t.Skip()
				}

				basename, _ := splitExt(entry.Name())

				var requests []testRequest

				extendedDir := filepath.Join(testdataDir, basename)
				if dirExists(extendedDir) {
					entries, err := os.ReadDir(extendedDir)
					if err != nil {
						t.Fatalf("reading extended dir: %v", err)
					}
					for _, entry := range entries {
						if strings.HasSuffix(entry.Name(), ".conf") {
							config := testRequest{name: entry.Name()}
							req, _ := splitExt(entry.Name())
							outFile := filepath.Join(extendedDir, req+".out")
							if !fileExists(outFile) {
								t.Fatalf("request file %v needs a matching output file", entry.Name())
							}
							reqFile := filepath.Join(extendedDir, entry.Name())
							{
								b, err := os.ReadFile(reqFile)
								if err != nil {
									t.Fatalf("reading request file %v: %v", reqFile, err)
								}
								s := string(b)
								lines := strings.Split(s, "\n")
								for _, line := range lines {
									line := strings.TrimSpace(line)
									if line != "" {
										pair := strings.SplitN(line, "=", 2)
										if len(pair) != 2 {
											t.Fatalf("illegal request key-value pair: %q", line)
										}
										switch pair[0] {
										case "requestPath":
											config.path = pair[1]
										case "queryParam":
											config.queryParams = append(config.queryParams, pair[1])
										default:
											log.Printf("unhandled request config key: %q", pair[0])
										}
									}
								}
							}
							{
								b, err := os.ReadFile(outFile)
								if err != nil {
									t.Fatalf("reading output file %v: %v", outFile, err)
								}
								config.expectedOutput = string(b)
							}
							requests = append(requests, config)
						}
					}
				}

				pushupFile := filepath.Join(testdataDir, entry.Name())

				outFile := filepath.Join(testdataDir, basename+".out")
				if _, err := os.Stat(outFile); err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						t.Fatalf("no matching output file %s", outFile)
					} else {
						t.Fatalf("stat'ing output file: %v", err)
					}
				}
				output, err := os.ReadFile(outFile)
				if err != nil {
					t.Fatalf("reading output file: %v", err)
				}

				// FIXME(paulsmith): add metadata to the testdata files with the
				// desired path to avoid these hacks
				// TODO(paulsmith): strip /testdata/ from request paths so tests
				// don't need to be aware of where they live
				requestPath := "/testdata/" + basename
				if basename == "index" {
					requestPath = "/testdata/"
				} else if basename == "$name" {
					requestPath = "/testdata/world"
				}

				requests = append(requests, testRequest{
					path:           requestPath,
					expectedOutput: string(output),
				})

				for _, request := range requests {
					t.Run(request.name, func(t *testing.T) {
						g, ctx0 := errgroup.WithContext(context.Background())
						ctx, cancel := context.WithTimeout(ctx0, 10*time.Second)
						defer cancel()

						ready := make(chan bool)
						done := make(chan bool)

						tmpdir, err := os.MkdirTemp("", "pushup")
						if err != nil {
							t.Fatalf("creating temp dir: %v", err)
						}
						defer os.RemoveAll(tmpdir)
						socketPath := filepath.Join(tmpdir, "sock")

						var errb bytes.Buffer
						var stdout io.ReadCloser
						var allgood bool
						var cmd *exec.Cmd

						g.Go(func() error {
							cmd = exec.Command(pushup, "run", "-page", pushupFile, "-unix-socket", socketPath)
							sysProcAttr(cmd)

							var err error
							stdout, err = cmd.StdoutPipe()
							if err != nil {
								return err
							}

							cmd.Stderr = &errb

							if err := cmd.Start(); err != nil {
								return err
							}

							if err := cmd.Wait(); err != nil {
								return err
							}

							return nil
						})

						g.Go(func() error {
							var buf [256]byte
							// NOTE(paulsmith): keep this in sync with the string in main.go
							needle := []byte("Pushup ready and listening on ")
							for {
								select {
								case <-ctx.Done():
									err := ctx.Err()
									return err
								default:
									if stdout != nil {
										for {
											n, err := stdout.Read(buf[:])
											if n > 0 {
												if bytes.Contains(buf[:], needle) {
													ready <- true
													return nil
												}
											} else {
												if errors.Is(err, io.EOF) {
													return nil
												}
												return err
											}
										}
									}
								}
							}
						})

						g.Go(func() error {
							select {
							case <-done:
								if cmd != nil {
									if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
										return err
									}
								}
								return nil
							case <-ctx.Done():
								err := ctx.Err()
								return err
							}
						})

						g.Go(func() error {
							select {
							case <-ready:
							case <-ctx.Done():
								err := ctx.Err()
								return err
							}
							client := &http.Client{
								Transport: &http.Transport{
									Dial: func(proto, addr string) (net.Conn, error) {
										return net.Dial("unix", socketPath)
									},
								},
							}
							var queryParams string
							if len(request.queryParams) > 0 {
								queryParams = "?" + strings.Join(request.queryParams, "&")
							}
							reqUrl := "http://dummy" + request.path + queryParams
							resp, err := client.Get(reqUrl)
							if err != nil {
								return nil
							}
							defer resp.Body.Close()
							got, err := io.ReadAll(resp.Body)
							if err != nil {
								return nil
							}
							done <- true
							if diff := cmp.Diff(request.expectedOutput, string(got)); diff != "" {
								t.Errorf("expected render diff (-want +got)\n%s", diff)
							} else {
								allgood = true
							}
							return nil
						})

						go func() {
							//nolint:errcheck
							g.Wait()
							close(ready)
							close(done)
						}()

						if err := g.Wait(); err != nil {
							if _, ok := err.(*exec.ExitError); ok && allgood {
								// no-op
							} else if _, ok := err.(*os.SyscallError); ok && allgood {
								// no-op
							} else {
								t.Logf("stderr:\n%s\n", errb.String())
								t.Fatalf("error: %T %v", err, err)
							}
						}
					})
				}
			})
		}
	}
}

func splitExt(path string) (name string, ext string) {
	ext = filepath.Ext(path)
	name = strings.TrimSuffix(path, ext)
	return
}

func TestTypenameFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"", ""},
		{"index", "Index"},
		{"$name", "DollarSignName"},
		{"default", "Default"},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := typenameFromPath(test.path)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}
