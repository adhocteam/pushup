package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
)

func TestPushup(t *testing.T) {
	samplesDir := "./samples"
	entries, err := os.ReadDir(samplesDir)
	if err != nil {
		t.Fatalf("reading samples dir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".pushup") {
			t.Run(entry.Name(), func(t *testing.T) {
				basename, _ := splitExt(entry.Name())
				requestPath := "/" + basename
				if basename == "index" {
					requestPath = "/"
				}
				pushupFile := filepath.Join(samplesDir, entry.Name())
				outFile := filepath.Join(samplesDir, basename+".out")
				if _, err := os.Stat(outFile); err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						t.Fatalf("no matching output file %s", outFile)
					} else {
						t.Fatalf("stat'ing output file: %v", err)
					}
				}

				want, err := os.ReadFile(outFile)
				if err != nil {
					t.Fatalf("reading output file: %v", err)
				}

				g, ctx0 := errgroup.WithContext(context.Background())
				ctx, cancel := context.WithTimeout(ctx0, 5*time.Second)
				defer cancel()

				ready := make(chan bool)
				done := make(chan bool)

				tmpdir, err := ioutil.TempDir("", "pushuptests")
				if err != nil {
					t.Fatalf("creating temp dir: %v", err)
				}
				defer os.RemoveAll(tmpdir)
				socketPath := filepath.Join(tmpdir, "pushup-"+strconv.Itoa(os.Getpid())+".sock")

				var errb bytes.Buffer

				g.Go(func() error {
					cmd := exec.Command("go", "run", "main.go", "-single", pushupFile, "-unix-socket", socketPath)
					cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

					stdout, err := cmd.StdoutPipe()
					if err != nil {
						return err
					}

					cmd.Stderr = &errb

					if err := cmd.Start(); err != nil {
						return err
					}

					g.Go(func() error {
						var buf [256]byte
						needle := []byte("Pushup ready and listening on ")
						select {
						case <-ctx.Done():
							err := ctx.Err()
							return err
						default:
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
					})

					g.Go(func() error {
						select {
						case <-done:
							syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
							cmd.Wait()
							return nil
						case <-ctx.Done():
							err := ctx.Err()
							return err
						}
					})

					if err := cmd.Wait(); err != nil {
						return err
					}

					return nil
				})

				var allgood bool

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
					resp, err := client.Get("http://dummy" + requestPath)
					if err != nil {
						return nil
					}
					defer resp.Body.Close()
					got, err := io.ReadAll(resp.Body)
					if err != nil {
						return nil
					}
					done <- true
					if diff := cmp.Diff(want, got); diff != "" {
						t.Errorf("expected render diff (-want +got)\n%s", diff)
					} else {
						allgood = true
					}
					return nil
				})

				go func() {
					g.Wait()
					close(ready)
					close(done)
				}()

				if err := g.Wait(); err != nil {
					if _, ok := err.(*exec.ExitError); ok && allgood {
						// no-op
					} else {
						t.Logf("stderr:\n%s\n", errb.String())
						t.Fatalf("error: %T %v", err, err)
					}
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

func TestTagString(t *testing.T) {
	tests := []struct {
		tag  tag
		want string
	}{
		{
			tag{name: "h1"},
			"h1",
		},
		{
			tag{name: "div", attr: []html.Attribute{{Key: "class", Val: "banner"}}},
			"div class=\"banner\"",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := test.tag.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}
