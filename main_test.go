package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
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

						tmpdir, err := ioutil.TempDir("", "pushuptests")
						if err != nil {
							t.Fatalf("creating temp dir: %v", err)
						}
						defer os.RemoveAll(tmpdir)
						socketPath := filepath.Join(tmpdir, "pushup-"+strconv.Itoa(os.Getpid())+"-"+strconv.Itoa(int(rand.Uint32()))+".sock")

						var errb bytes.Buffer
						var stdout io.ReadCloser
						var allgood bool
						var cmd *exec.Cmd

						g.Go(func() error {
							cmd = exec.Command(pushup, "run", "-build-pkg", "github.com/adhocteam/pushup/build", "-page", pushupFile, "-unix-socket", socketPath)
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
									syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
									cmd.Wait()
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
			tag{name: "div", attrs: []*attr{{name: stringPos{string: "class"}, value: stringPos{string: "banner"}}}},
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

func TestRouteForPage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			"index.up",
			"/",
		},
		{
			"about.up",
			"/about",
		},
		{
			"x/sub.up",
			"/x/sub",
		},
		{
			"testdata/foo.up",
			"/testdata/foo",
		},
		{
			"x/$name.up",
			"/x/:name",
		},
		{
			"$projectId/$productId",
			"/:projectId/:productId",
		},
		{
			"blah/index.up",
			"/blah/",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := routeForPage(test.path); test.want != got {
				t.Errorf("want %q, got %q", test.want, got)
			}
		})
	}
}

func TestCompiledOutputPath(t *testing.T) {
	tests := []struct {
		pfile    projectFile
		want     string
		strategy upFileType
	}{
		{
			projectFile{path: "app/pages/index.up", projectFilesSubdir: "app/pages"},
			"index.up.go",
			upFilePage,
		},
		{
			projectFile{path: "app/pages/about.up", projectFilesSubdir: "app/pages"},
			"about.up.go",
			upFilePage,
		},
		{
			projectFile{path: "app/pages/x/sub.up", projectFilesSubdir: "app/pages"},
			"x__sub.up.go",
			upFilePage,
		},
		{
			projectFile{path: "testdata/foo.up", projectFilesSubdir: ""},
			"testdata__foo.up.go",
			upFilePage,
		},
		{
			projectFile{path: "app/layouts/default.up", projectFilesSubdir: "app/layouts"},
			"default.layout.up.go",
			upFileLayout,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := compiledOutputPath(test.pfile, test.strategy); test.want != got {
				t.Errorf("want %q, got %q", test.want, got)
			}
		})
	}
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

func TestGeneratedTypename(t *testing.T) {
	tests := []struct {
		pfile    projectFile
		strategy upFileType
		want     string
	}{
		{projectFile{path: "index.up", projectFilesSubdir: "."}, upFilePage, "IndexPage"},
		{projectFile{path: "foo-bar.up", projectFilesSubdir: "."}, upFilePage, "FooBarPage"},
		{projectFile{path: "foo_bar.up", projectFilesSubdir: "."}, upFilePage, "FooBarPage"},
		{projectFile{path: "a/b/c.up", projectFilesSubdir: "."}, upFilePage, "ABCPage"},
		{projectFile{path: "a/b/$c.up", projectFilesSubdir: "."}, upFilePage, "ABDollarSignCPage"},
	}

	for _, test := range tests {
		t.Run(test.pfile.path, func(t *testing.T) {
			got := generatedTypename(test.pfile, test.strategy)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}

func TestOpenTagLexer(t *testing.T) {
	tests := []struct {
		input string
		want  []*attr
	}{
		{
			"<div>",
			[]*attr{},
		},
		{
			"<div disabled>",
			[]*attr{{name: stringPos{"disabled", pos(5)}}},
		},
		{
			`<div class="foo">`,
			[]*attr{{name: stringPos{"class", pos(5)}, value: stringPos{"foo", pos(12)}}},
		},
		{
			`<p   data-^name="/foo/bar/^value"   thing="^asd"  >`,
			[]*attr{
				{
					name: stringPos{
						"data-^name",
						pos(5),
					},
					value: stringPos{
						"/foo/bar/^value",
						pos(17),
					},
				},
				{
					name: stringPos{
						"thing",
						pos(36),
					},
					value: stringPos{
						"^asd",
						pos(43),
					},
				},
			},
		},
	}
	opts := cmp.AllowUnexported(attr{}, stringPos{})
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := scanAttrs(tt.input)
			if err != nil {
				t.Fatalf("scanAttrs: %v", err)
			}
			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		input string
		want  *syntaxTree
	}{
		// escaped transition symbol
		{
			`^^foo`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "^", pos: span{start: 0, end: 2}},
					&nodeLiteral{str: "foo", pos: span{start: 2, end: 5}},
				},
			},
		},
		{
			`<a href="^^foo"></a>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<a ", pos: span{end: 3}},
					&nodeLiteral{str: "href", pos: span{start: 3, end: 7}},
					&nodeLiteral{str: `="`, pos: span{start: 7, end: 9}},
					&nodeLiteral{str: "^", pos: span{start: 9, end: 10}},
					&nodeLiteral{str: `foo">`, pos: span{start: 11, end: 16}},
					&nodeLiteral{str: "</a>", pos: span{start: 16, end: 20}},
				},
			},
		},
		{
			`<p>Hello, ^name!</p>
^{
	name := "world"
}
`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<p>", pos: span{start: 0, end: 3}},
					&nodeLiteral{str: "Hello, ", pos: span{start: 3, end: 10}},
					&nodeGoStrExpr{expr: "name", pos: span{start: 11, end: 15}},
					&nodeLiteral{str: "!", pos: span{start: 15, end: 16}},
					&nodeLiteral{str: "</p>", pos: span{start: 16, end: 20}},
					&nodeLiteral{str: "\n", pos: span{start: 20, end: 21}},
					&nodeGoCode{code: "name := \"world\"\n", pos: span{start: 23, end: 39}},
					&nodeLiteral{str: "\n", pos: span{start: 42, end: 43}},
				},
			},
		},
		{
			`^if name != "" {
	<h1>Hello, ^name!</h1>
} ^else {
	<h1>Hello, world!</h1>
}`,
			&syntaxTree{
				nodes: []node{
					&nodeIf{
						cond: &nodeGoStrExpr{
							expr: "name != \"\"",
							pos:  span{start: 4, end: 14},
						},
						then: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n\t", pos: span{start: 16, end: 18}},
								&nodeElement{
									tag:           tag{name: "h1"},
									startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 18, end: 22}}},
									pos:           span{start: 18, end: 22},
									children: []node{
										&nodeLiteral{str: "Hello, ", pos: span{start: 22, end: 29}},
										&nodeGoStrExpr{expr: "name", pos: span{start: 30, end: 34}},
										&nodeLiteral{str: "!", pos: span{start: 34, end: 35}},
									},
								},
							},
						},
						alt: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n\t", pos: span{start: 50, end: 52}},
								&nodeElement{
									tag:           tag{name: "h1"},
									startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 52, end: 56}}},
									pos:           span{start: 52, end: 56},
									children: []node{
										&nodeLiteral{str: "Hello, world!", pos: span{start: 56, end: 69}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			`^if name == "" {
    <div>
        <h1>Hello, world!</h1>
    </div>
} ^else {
    <div>
        <h1>Hello, ^name!</h1>
    </div>
}`,
			&syntaxTree{
				nodes: []node{
					&nodeIf{
						cond: &nodeGoStrExpr{
							expr: "name == \"\"",
							pos:  span{start: 4, end: 14},
						},
						then: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n    ", pos: span{start: 16, end: 21}},
								&nodeElement{
									tag:           tag{name: "div"},
									startTagNodes: []node{&nodeLiteral{str: "<div>", pos: span{start: 21, end: 26}}},
									pos:           span{start: 21, end: 26},
									children: []node{
										&nodeLiteral{str: "\n        ", pos: span{start: 26, end: 35}},
										&nodeElement{
											tag:           tag{name: "h1"},
											startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 35, end: 39}}},
											pos:           span{start: 35, end: 39},
											children: []node{
												&nodeLiteral{str: "Hello, world!", pos: span{start: 39, end: 52}},
											},
										},
										&nodeLiteral{str: "\n    ", pos: span{start: 57, end: 62}},
									},
								},
							},
						},
						alt: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n    ", pos: span{start: 78, end: 83}},
								&nodeElement{
									tag:           tag{name: "div"},
									pos:           span{start: 83, end: 88},
									startTagNodes: []node{&nodeLiteral{str: "<div>", pos: span{start: 83, end: 88}}},
									children: []node{
										&nodeLiteral{str: "\n        ", pos: span{start: 88, end: 97}},
										&nodeElement{
											tag:           tag{name: "h1"},
											startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 97, end: 101}}},
											pos:           span{start: 97, end: 101},
											children: []node{
												&nodeLiteral{str: "Hello, ", pos: span{start: 101, end: 108}},
												&nodeGoStrExpr{expr: "name", pos: span{start: 109, end: 113}},
												&nodeLiteral{str: "!", pos: span{start: 113, end: 114}},
											},
										},
										&nodeLiteral{str: "\n    ", pos: span{start: 119, end: 124}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			`^{
	type product struct {
		name string
		price float32
	}

	products := []product{{name: "Widget", price: 9.49}}
}`,
			&syntaxTree{
				nodes: []node{
					&nodeGoCode{
						code: `type product struct {
		name string
		price float32
	}

	products := []product{{name: "Widget", price: 9.49}}
`,
						pos: span{start: 2, end: 112},
					},
				},
			},
		},
		{
			`^layout !`,
			&syntaxTree{
				nodes: []node{
					&nodeLayout{name: "!", pos: span{start: 1, end: 9}},
				},
			},
		},
		{
			`^import "time"`,
			&syntaxTree{
				nodes: []node{
					&nodeImport{
						decl: importDecl{
							pkgName: "",
							path:    "\"time\"",
						},
						pos: span{},
					},
				},
			},
		},
		{
			`<a href="^url">x</a>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<a ", pos: span{start: 0, end: 3}},
					&nodeLiteral{str: "href", pos: span{start: 3, end: 7}},
					&nodeLiteral{str: "=\"", pos: span{start: 7, end: 9}},
					&nodeGoStrExpr{expr: "url", pos: span{start: 0, end: 3}}, // FIXME(paulsmith): should be 9 and 13 but this part of the parser is not correct yet
					&nodeLiteral{str: "\">", pos: span{start: 13, end: 15}},
					&nodeLiteral{str: "x", pos: span{start: 15, end: 16}},
					&nodeLiteral{str: "</a>", pos: span{start: 16, end: 20}},
				},
			},
		},
		{
			`^if true { <a href="^url"><div data-^foo="bar"></div></a> }`,
			&syntaxTree{
				nodes: []node{
					&nodeIf{
						cond: &nodeGoStrExpr{expr: "true", pos: span{start: 4, end: 8}},
						then: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: " ", pos: span{start: 10, end: 11}},
								&nodeElement{
									tag: tag{
										name: "a",
										attrs: []*attr{
											{
												name:  stringPos{string: "href", start: pos(3)},
												value: stringPos{string: "^url", start: pos(9)},
											},
										},
									},
									startTagNodes: []node{
										&nodeLiteral{str: "<a ", pos: span{start: 11, end: 14}},
										&nodeLiteral{str: "href", pos: span{start: 14, end: 18}},
										&nodeLiteral{str: `="`, pos: span{start: 18, end: 20}},
										&nodeGoStrExpr{expr: "url", pos: span{end: 3}},
										&nodeLiteral{str: `">`, pos: span{start: 24, end: 26}},
									},
									children: []node{
										&nodeElement{
											tag: tag{
												name: "div",
												attrs: []*attr{
													{
														name:  stringPos{string: "data-^foo", start: pos(5)},
														value: stringPos{string: "bar", start: pos(16)},
													},
												},
											},
											startTagNodes: []node{
												&nodeLiteral{str: "<div ", pos: span{start: 26, end: 31}},
												&nodeLiteral{str: "data-", pos: span{start: 31, end: 36}},
												&nodeGoStrExpr{expr: "foo", pos: span{start: 0, end: 3}}, // FIXME
												&nodeLiteral{str: "=\"", pos: span{start: 40, end: 42}},
												&nodeLiteral{str: "bar", pos: span{start: 42, end: 45}},
												&nodeLiteral{str: "\">", pos: span{start: 45, end: 47}},
											},
											pos: span{start: 26, end: 47},
										},
									},
									pos: span{start: 11, end: 26},
								},
							},
						},
					},
				},
			},
		},
		{
			`^section foo {<text>bar</text>}`,
			&syntaxTree{
				nodes: []node{
					&nodeSection{
						name: "foo",
						pos:  span{start: 8, end: 12},
						block: &nodeBlock{
							nodes: []node{
								&nodeBlock{nodes: []node{&nodeLiteral{str: "bar", pos: span{start: 20, end: 23}}}},
							},
						},
					},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^foo.bar("asd").baz.biz()`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `foo.bar("asd").baz.biz()`, pos: span{start: 1, end: 25}},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^quux[42]`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `quux[42]`, pos: span{start: 1, end: 9}},
				},
			},
		},
		{
			// space separates the Go ident `name` from the next `(`, ending the
			// expression and preventing it from being interpreted as a function
			// call
			`^p.name ($blah ...`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `p.name`, pos: span{start: 1, end: 7}},
					&nodeLiteral{str: ` ($blah ...`, pos: span{start: 7, end: 18}},
				},
			},
		},
		{
			// expanded implicit/simple expression with argument list in func call
			`^getParam(req, "name")`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `getParam(req, "name")`, pos: span{start: 1, end: 22}},
				},
			},
		},
		{
			`^partial foo {<text>bar</text>}`,
			&syntaxTree{
				nodes: []node{
					&nodePartial{
						name: "foo",
						pos:  span{start: 8, end: 12},
						block: &nodeBlock{
							nodes: []node{
								&nodeBlock{nodes: []node{&nodeLiteral{str: "bar", pos: span{start: 20, end: 23}}}},
							},
						},
					},
				},
			},
		},
		{
			`^partial foo {
				^if true {
					<p></p>
				}
			}`,
			&syntaxTree{
				nodes: []node{
					&nodePartial{
						name: "foo",
						pos:  span{start: 8, end: 12},
						block: &nodeBlock{
							nodes: []node{
								&nodeIf{
									cond: &nodeGoStrExpr{expr: "true", pos: span{23, 27}},
									then: &nodeBlock{
										nodes: []node{
											&nodeLiteral{str: "\n\t\t\t\t\t", pos: span{start: 29, end: 35}},
											&nodeElement{
												tag:           tag{name: "p"},
												startTagNodes: []node{&nodeLiteral{str: "<p>", pos: span{start: 35, end: 38}}},
												pos:           span{35, 38},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			`My name is ^name. What's yours?`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "My name is ", pos: span{end: 11}},
					&nodeGoStrExpr{expr: "name", pos: span{start: 12, end: 16}},
					&nodeLiteral{str: ". What's yours?", pos: span{start: 16, end: 31}},
				},
			},
		},
		{
			`<p>^foo.</p>`,
			&syntaxTree{
				nodes: []node{
					&nodeLiteral{str: "<p>", pos: span{end: 3}},
					&nodeGoStrExpr{expr: "foo", pos: span{start: 4, end: 7}},
					&nodeLiteral{str: ".", pos: span{start: 7, end: 8}},
					&nodeLiteral{str: "</p>", pos: span{start: 8, end: 12}},
				},
			},
		},
	}
	opts := cmp.AllowUnexported(unexported...)
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := parse(test.input)
			if err != nil {
				t.Fatalf("unexpected error parsing input: %v", err)
			}
			if diff := cmp.Diff(test.want, got, opts); diff != "" {
				t.Errorf("expected parse diff (-want +got):\n%s", diff)
			}
		})
	}
}

func TestParseSyntaxErrors(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"^if"},
		// FIXME(paulsmith): add more syntax errors
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			tree, err := parse(tt.input)
			if tree != nil {
				t.Errorf("expected nil tree, got %v", tree)
			}
			if err == nil {
				t.Errorf("expected parse error, got nil")
			}
		})
	}
}

var unexported = []any{
	attr{},
	importDecl{},
	nodeBlock{},
	nodeElement{},
	nodeGoCode{},
	nodeGoStrExpr{},
	nodeIf{},
	nodeImport{},
	nodeLayout{},
	nodeLiteral{},
	nodeSection{},
	nodePartial{},
	span{},
	stringPos{},
	syntaxTree{},
	tag{},
}

func TestCodeGenFromNode(t *testing.T) {
	tests := []struct {
		node node
		want string
	}{
		{
			node: &nodeElement{
				tag: tag{name: "div", attrs: []*attr{{name: stringPos{string: "id"}, value: stringPos{string: "foo"}}}},
				startTagNodes: []node{
					&nodeLiteral{str: "<div "},
					&nodeLiteral{str: "id=\""},
					&nodeLiteral{str: "foo"},
					&nodeLiteral{str: "\">"},
				},
				children: []node{&nodeLiteral{str: "bar"}},
			},
			want: `io.WriteString(w, "<div ")
io.WriteString(w, "id=\"")
io.WriteString(w, "foo")
io.WriteString(w, "\">")
io.WriteString(w, "bar")
io.WriteString(w, "</div>")
`,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			page, err := newPageFromTree(&syntaxTree{nodes: []node{test.node}})
			if err != nil {
				t.Fatalf("new page from tree: %v", err)
			}
			g := newPageCodeGen(page, projectFile{}, "")
			g.lineDirectivesEnabled = false
			g.genNode(test.node)
			got := g.bodyb.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("expected code gen diff (-want +got):\n%s", diff)
			}
		})
	}
}

func FuzzParser(f *testing.F) {
	seeds := []string{
		"",
		"^layout !\n",
		"<h1>Hello, world!</h1>",
		"^{ name := \"world\" }\n<h1>Hello, ^name!</h1>\n",
		"^if true {\n<a href=\"^req.URL.Path\">this page</a>\n}\n",
		"<div>^(3 + 4 * 5)</div>\n",
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, in []byte) {
		_, err := parse(string(in))
		if err != nil {
			if _, ok := err.(syntaxError); !ok {
				t.Errorf("expected syntax error, got %T %v", err, err)
			}
		}
	})
}

func FuzzOpenTagLexer(f *testing.F) {
	seeds := []string{
		"<a href=\"https://adhoc.team/\">",
		"<b>",
		"<input checked>",
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, in []byte) {
		_, err := scanAttrs(string(in))
		if err != nil {
			if _, ok := err.(openTagScanError); !ok {
				t.Errorf("expected scan error, got %T %v", err, err)
			}
		}
	})
}
