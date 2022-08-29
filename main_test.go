package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
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

func TestPushup(t *testing.T) {
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
		if strings.HasSuffix(entry.Name(), ".pushup") {
			t.Run(entry.Name(), func(t *testing.T) {
				basename, _ := splitExt(entry.Name())
				// FIXME(paulsmith): add metadata to the testdata files with the
				// desired path to avoid these hacks
				requestPath := "/testdata/" + basename
				if basename == "index" {
					requestPath = "/testdata/"
				} else if basename == "$name" {
					requestPath = "/testdata/world"
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
				socketPath := filepath.Join(tmpdir, "pushup-"+strconv.Itoa(os.Getpid())+"-"+strconv.Itoa(int(rand.Uint32()))+".sock")

				var errb bytes.Buffer

				g.Go(func() error {
					cmd := exec.Command(pushup, "run", "-build-pkg", "github.com/AdHocRandD/pushup/build", "-page", pushupFile, "-unix-socket", socketPath)
					sysProcAttr(cmd)

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
						// NOTE(paulsmith): keep this in sync with the string in main.go
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
							syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
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
					if diff := cmp.Diff(string(want), string(got)); diff != "" {
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

func TestTrimCommonPrefix(t *testing.T) {
	tests := []struct {
		path string
		root string
		want string
	}{
		{
			"app/pages/index.pushup",
			"app/pages",
			"index.pushup",
		},
		{
			"./app/pages/index.pushup",
			"app/pages",
			"index.pushup",
		},
		{
			"index.pushup",
			".",
			"index.pushup",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := trimCommonPrefix(test.path, test.root); test.want != got {
				t.Errorf("want %q, got %q", test.want, got)
			}
		})
	}
}

func TestRouteFromPath(t *testing.T) {
	tests := []struct {
		path string
		root string
		want string
	}{
		{
			"app/pages/index.pushup",
			"app/pages",
			"/",
		},
		{
			"app/pages/about.pushup",
			"app/pages",
			"/about",
		},
		{
			"app/pages/x/sub.pushup",
			"app/pages",
			"/x/sub",
		},
		{
			"testdata/foo.pushup",
			".",
			"/testdata/foo",
		},
		{
			"app/pages/x/$name.pushup",
			"app/pages",
			"/x/:name",
		},
		{
			"app/pages/$projectId/$productId",
			"app/pages",
			"/:projectId/:productId",
		},
		{
			"app/pages/blah/index.pushup",
			"app/pages",
			"/blah/",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := routeFromPath(test.path, test.root); test.want != got {
				t.Errorf("want %q, got %q", test.want, got)
			}
		})
	}
}

func TestGeneratedFilename(t *testing.T) {
	tests := []struct {
		path     string
		root     string
		want     string
		strategy compilationStrategy
	}{
		{
			"app/pages/index.pushup",
			"app/pages",
			"index.up.go",
			compilePushupPage,
		},
		{
			"app/pages/about.pushup",
			"app/pages",
			"about.up.go",
			compilePushupPage,
		},
		{
			"app/pages/x/sub.pushup",
			"app/pages",
			"x__sub.up.go",
			compilePushupPage,
		},
		{
			"testdata/foo.pushup",
			".",
			"testdata__foo.up.go",
			compilePushupPage,
		},
		{
			"app/layouts/default.pushup",
			"app/layouts",
			"default.layout.up.go",
			compileLayout,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			if got := generatedFilename(test.path, test.root, test.strategy); test.want != got {
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
		path     string
		root     string
		strategy compilationStrategy
		want     string
	}{
		{"index.pushup", ".", compilePushupPage, "IndexPage"},
		{"foo-bar.pushup", ".", compilePushupPage, "FooBarPage"},
		{"foo_bar.pushup", ".", compilePushupPage, "FooBarPage"},
		{"a/b/c.pushup", ".", compilePushupPage, "ABCPage"},
		{"a/b/$c.pushup", ".", compilePushupPage, "ABDollarSignCPage"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			got := generatedTypename(test.path, test.root, test.strategy)
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
			nil,
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
			l := newOpenTagLexer(tt.input)
			got := l.scan()
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
				}},
		},
		{
			`^if name != "" {
	<h1>Hello, ^name!</h1>
} else {
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
									}},
							},
						},
						alt: &nodeBlock{
							nodes: []node{
								&nodeLiteral{str: "\n\t", pos: span{start: 49, end: 51}},
								&nodeElement{
									tag:           tag{name: "h1"},
									startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 51, end: 55}}},
									pos:           span{start: 51, end: 55},
									children: []node{
										&nodeLiteral{str: "Hello, world!", pos: span{start: 55, end: 68}},
									}},
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
} else {
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
								&nodeLiteral{str: "\n    ", pos: span{start: 77, end: 82}},
								&nodeElement{
									tag:           tag{name: "div"},
									pos:           span{start: 82, end: 87},
									startTagNodes: []node{&nodeLiteral{str: "<div>", pos: span{start: 82, end: 87}}},
									children: []node{
										&nodeLiteral{str: "\n        ", pos: span{start: 87, end: 96}},
										&nodeElement{
											tag:           tag{name: "h1"},
											startTagNodes: []node{&nodeLiteral{str: "<h1>", pos: span{start: 96, end: 100}}},
											pos:           span{start: 96, end: 100},
											children: []node{
												&nodeLiteral{str: "Hello, ", pos: span{start: 100, end: 107}},
												&nodeGoStrExpr{expr: "name", pos: span{start: 108, end: 112}},
												&nodeLiteral{str: "!", pos: span{start: 112, end: 113}},
											},
										},
										&nodeLiteral{str: "\n    ", pos: span{start: 118, end: 123}},
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
						pos: span{}},
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
					&nodeGoStrExpr{expr: `foo.bar("asd").baz.biz()`, pos: span{start: 1, end: 24}},
				},
			},
		},
		{
			// example of expanded implicit/simple expression
			`^quux[42]`,
			&syntaxTree{
				nodes: []node{
					&nodeGoStrExpr{expr: `quux[42]`, pos: span{start: 1, end: 8}},
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
					&nodeGoStrExpr{expr: `getParam(req, "name")`, pos: span{start: 1, end: 21}},
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
	span{},
	stringPos{},
	syntaxTree{},
	tag{},
}
