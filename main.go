package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var outDir = "./build"

var singleFlag = flag.String("single", "", "path to a single Pushup file")
var applyOptimizations = flag.Bool("O", false, "apply simple optimizations to the parse tree")

func main() {
	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	parseOnly := flag.Bool("parse-only", false, "exit after dumping parse result")
	compileOnly := flag.Bool("compile-only", false, "compile only, don't start web server after")

	flag.Parse()

	var layoutsDir string
	var pagesDir string

	var layoutFiles []string
	var pushupFiles []string

	os.RemoveAll(outDir)

	appDir := "app"
	if flag.NArg() == 1 {
		appDir = flag.Arg(0)
	}

	if *singleFlag != "" {
		pushupFiles = []string{*singleFlag}
	} else {
		layoutsDir = filepath.Join(appDir, "layouts")
		{
			if !dirExists(layoutsDir) {
				log.Fatalf("invalid Pushup project directory structure: couldn't find `layouts` subdir")
			}

			entries, err := os.ReadDir(layoutsDir)
			if err != nil {
				log.Fatalf("reading app directory: %v", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pushup") {
					path := filepath.Join(layoutsDir, entry.Name())
					layoutFiles = append(layoutFiles, path)
				}
			}
		}

		pagesDir = filepath.Join(appDir, "pages")
		{
			if !dirExists(pagesDir) {
				log.Fatalf("invalid Pushup project directory structure: couldn't find `pages` subdir")
			}

			pushupFiles = getPushupPagePaths(pagesDir)
		}
	}

	if *parseOnly {
		for _, path := range pushupFiles {
			b, err := os.ReadFile(path)
			if err != nil {
				log.Fatalf("reading file %s: %v", path, err)
			}
			tree, err := parse(string(b))
			if err != nil {
				log.Fatalf("parsing file %s: %v", path, err)
			}
			(&debugPrettyPrinter{w: os.Stdout}).visitNodes(tree.nodes)
		}
		os.Exit(0)
	}

	for _, path := range layoutFiles {
		if err := compilePushup(path, layoutsDir, compileLayout, outDir); err != nil {
			log.Fatalf("compiling layout file %s: %v", path, err)
		}
	}

	for _, path := range pushupFiles {
		if err := compilePushup(path, pagesDir, compilePushupPage, outDir); err != nil {
			log.Fatalf("compiling pushup file %s: %v", path, err)
		}
	}

	addSupport(outDir)

	if !*compileOnly {
		var args []string
		if *unixSocket != "" {
			args = []string{"-unix-socket", *unixSocket}
		} else {
			args = []string{"-port", *port}
		}
		// FIXME(paulsmith): separate build from run and move it in to compile step
		if err := buildAndRun(outDir, args); err != nil {
			log.Fatalf("building and running generated Go code: %v", err)
		}
	}
}

func getPushupPagePaths(root string) []string {
	var paths []string
	err := fs.WalkDir(os.DirFS(root), ".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && filepath.Ext(path) == ".pushup" {
			paths = append(paths, filepath.Join(root, path))
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return paths
}

func addSupport(dir string) {
	supportFile := `
package build

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
)

// FIXME(paulsmith): I think of this as a route but this conflicts with a route in the serve mux
// sense, so calling "component" for now
type component interface {
	// FIXME(paulsmith): return a pushup.Response object instead and don't take a writer
	Render(io.Writer, *http.Request) error
}
// FIXME(paulsmith): add a wrapper type for easily going between a component and a http.Handler

type routeList []route

var routes routeList

func (r *routeList) add(path string, c component) {
	*r = append(*r, newRoute(path, c))
}

type route struct {
	path string
	regex *regexp.Regexp
	component component
}

func newRoute(path string, c component) route {
	return route{path, regexp.MustCompile("^" + path + "$"), c}
}

var NotFound = errors.New("page not found")

func Render(w http.ResponseWriter, r *http.Request) error {
	for _, route := range routes {
		matches := route.regex.FindStringSubmatch(r.URL.Path)
		if len(matches) > 0 {
			// TODO(paulsmith): implement matches
			if err := route.component.Render(w, r); err != nil {
				return err
			}
			return nil
		}
	}
	return NotFound
}

type layout interface {
	Render(yield chan struct{}, w io.Writer, req *http.Request) error
}

var layouts = make(map[string]layout)

func getLayout(name string) layout {
	l, ok := layouts[name]
	if !ok {
		panic("couldn't find layout " + name)
	}
	return l
}

func Admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<h1>Routes</h1>\n<ul>\n")
	for _, route := range routes {
		fmt.Fprintf(w, "\t<li>%s</li>\n", route.path)
	}
	fmt.Fprintf(w, "</ul>\n")
}
`
	if err := os.WriteFile(filepath.Join(dir, "pushup_support.go"), []byte(supportFile), 0644); err != nil {
		panic(err)
	}
}

func buildAndRun(dir string, passthruArgs []string) error {
	mainExeDir := filepath.Join(dir, "cmd", "myproject")
	if err := os.MkdirAll(mainExeDir, 0755); err != nil {
		return fmt.Errorf("making directory for command: %w", err)
	}

	mainProgram := `package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/AdHocRandD/pushup/build"

	"golang.org/x/sync/errgroup"
)

var logger *log.Logger

func main() {
	// FIXME(paulsmith): detect if connected to terminal for VT100 escapes
	logger = log.New(os.Stderr, "[\x1b[36mPUSHUP\x1b[0m] ", 0)

	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	adminPort := flag.String("admin-port", "9090", "port to listen on for admin")

	// FIXME(paulsmith): can't have both port and unixSocket non-empty
	flag.Parse()

	g := new(errgroup.Group)

	g.Go(func() error {
		mux := http.NewServeMux()
		// TODO(paulsmith): allow these middlewares to be configurable on/off
		mux.Handle("/", panicRecoveryMiddleware(requestLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := build.Render(w, r); err != nil {
				logger.Printf("rendering route: %v", err)
				if errors.Is(err, build.NotFound) {
					http.NotFound(w, r)
				} else {
					http.Error(w, http.StatusText(500), 500)
				}
				return
			}
		}))))

		var ln net.Listener
		var err error
		if *unixSocket != "" {
			ln, err = net.Listen("unix", *unixSocket)
		} else {
			host := "0.0.0.0"
			addr := host + ":" + *port
			ln, err = net.Listen("tcp4", addr) // TODO(paulsmith): may want to support IPv6
		}
		if err != nil {
			return fmt.Errorf("getting a listener: %v", err)
		}

		srv := http.Server {Handler:mux}

		fmt.Fprintf(os.Stdout, "\x1b[32m↑↑ Pushup ready and listening on %s ↑↑\x1b[0m\n", ln.Addr().String())
		if err := srv.Serve(ln); err != nil {
			return fmt.Errorf("serving HTTP: %v", err)
		}

		return nil
	})

	g.Go(func() error {
		mux := http.NewServeMux()
		mux.Handle("/", http.HandlerFunc(build.Admin))
		srv := http.Server {
			Addr: "127.0.0.1:" + *adminPort,
			Handler: mux,
		}
		return srv.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		logger.Fatalf("error: %v", err)
	}
}

func panicRecoveryMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("recovered from panic in an HTTP hander: %v", r)
				debug.PrintStack()
				http.Error(w, http.StatusText(500), 500)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	code int
	wrote bool
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, 200, false}
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
	return
}

func (w *loggingResponseWriter) Flush() {
	if fl, ok := w.ResponseWriter.(http.Flusher); ok {
		if w.code == 0 {
			w.WriteHeader(200)
		}
		fl.Flush()
	}
}

func requestLogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t0 := time.Now()
		lwr := newLoggingResponseWriter(w)
		h.ServeHTTP(lwr, r)
		logger.Printf("%s %s %d %s", r.Method, r.URL.String(), lwr.code, time.Since(t0))
	})
}
`

	if err := os.WriteFile(filepath.Join(mainExeDir, "main.go"), []byte(mainProgram), 0644); err != nil {
		return fmt.Errorf("writing main.go file: %w", err)
	}

	args := append([]string{"run", "./build/cmd/myproject"}, passthruArgs...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running project main executable: %w", err)
	}

	return nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false
		}
		panic(err)
	}
	return fi.IsDir()
}

type compilationStrategy int

const (
	compilePushupPage compilationStrategy = iota
	compileLayout
)

func compilePushup(sourcePath string, rootDir string, strategy compilationStrategy, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	b, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	source := string(b)

	tree, err := parse(source)
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic while parsing %s: %v", sourcePath, err)
			panic(err)
		}
	}()

	// apply some simple optimizations
	if *applyOptimizations {
		tree = optimize(tree)
	}

	var c codeGenUnit
	switch strategy {
	case compilePushupPage:
		page, err := postProcessTree(tree)
		if err != nil {
			return fmt.Errorf("post-processing tree: %w", err)
		}
		layoutName := page.layout
		if *singleFlag != "" {
			layoutName = ""
		}
		route := routeFromPath(sourcePath, rootDir)
		c = &pageCodeGen{source: source, layout: layoutName, page: page, route: route}
	case compileLayout:
		c = &layoutCodeGen{source: source, tree: tree}
	default:
		panic("")
	}

	outputFilename := generatedFilename(sourcePath, rootDir, strategy)
	outputPath := filepath.Join(targetDir, outputFilename)
	basename := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))

	if err := generateCodeToFile(c, basename, outputPath, strategy); err != nil {
		return fmt.Errorf("generating Go code from parse result: %w", err)
	}

	return nil
}

// generatedFilename returns the filename for the .go file containing the
// generated code for the Pushup page.
func generatedFilename(path string, root string, strategy compilationStrategy) string {
	path = trimCommonPrefix(path, root)
	var dirs []string
	dir := filepath.Dir(path)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(path)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	prefix := strings.Join(dirs, "__")
	var result string
	var suffix string
	switch strategy {
	case compileLayout:
		suffix = "_layout"
	case compilePushupPage:
		suffix = ""
	default:
		panic("")
	}
	if prefix != "" {
		result = prefix + "__" + base + suffix + ".go"
	} else {
		result = base + suffix + ".go"
	}
	return result
}

// routeFromPath produces the URL path route from the name of the Pushup page.
// path is the path to the Pushup file. root is the path of the root of the
// Pushup project.
func routeFromPath(path string, root string) string {
	path = trimCommonPrefix(path, root)
	var dirs []string
	dir := filepath.Dir(path)
	if dir != "." {
		dirs = strings.Split(dir, string([]rune{os.PathSeparator}))
	}
	file := filepath.Base(path)
	base := strings.TrimSuffix(file, filepath.Ext(file))
	var route string
	if base == "index" {
		route = "/" + strings.Join(dirs, "/")
	} else {
		route = "/" + strings.Join(append(dirs, base), "/")
	}
	return route
}

func trimCommonPrefix(path string, prefix string) string {
	path = filepath.Clean(path)
	prefix = filepath.Clean(prefix)
	stripped := strings.TrimPrefix(path, prefix)
	if strings.HasPrefix(stripped, "/") {
		stripped = strings.TrimPrefix(stripped, "/")
	}
	return stripped
}

// node represents a portion of the Pushup syntax, like a chunk of HTML,
// or a Go expression to be evaluated, or a control flow construct like `if'
// or `for'.
type node interface {
	Pos() span
	nodeAcceptor
}

type nodeAcceptor interface {
	accept(nodeVisitor)
}

type nodeList []node

func (n nodeList) Pos() span            { return n[0].Pos() }
func (n nodeList) accept(v nodeVisitor) { v.visitNodes(n) }

type nodeVisitor interface {
	visitElement(*nodeElement)
	visitLiteral(*nodeLiteral)
	visitGoStrExpr(*nodeGoStrExpr)
	visitGoCode(*nodeGoCode)
	visitIf(*nodeIf)
	visitFor(*nodeFor)
	visitStmtBlock(*nodeBlock)
	visitNodes([]node)
	visitImport(*nodeImport)
	visitLayout(*nodeLayout)
}

type nodeLiteral struct {
	str string
	pos span
}

func (e nodeLiteral) Pos() span             { return e.pos }
func (e *nodeLiteral) accept(v nodeVisitor) { v.visitLiteral(e) }

var _ node = (*nodeLiteral)(nil)

type nodeGoStrExpr struct {
	expr string
	pos  span
}

func (e nodeGoStrExpr) Pos() span             { return e.pos }
func (e *nodeGoStrExpr) accept(v nodeVisitor) { v.visitGoStrExpr(e) }

var _ node = (*nodeGoStrExpr)(nil)

func newGoStrExpr(expr string, start, end int) *nodeGoStrExpr {
	return &nodeGoStrExpr{expr, span{start, end}}
}

type nodeGoCode struct {
	code string
	pos  span
}

func (e nodeGoCode) Pos() span             { return e.pos }
func (e *nodeGoCode) accept(v nodeVisitor) { v.visitGoCode(e) }

var _ node = (*nodeGoCode)(nil)

type nodeIf struct {
	cond *nodeGoStrExpr
	then *nodeBlock
	alt  *nodeBlock
}

func (e nodeIf) Pos() span             { return e.cond.pos }
func (e *nodeIf) accept(v nodeVisitor) { v.visitIf(e) }

var _ node = (*nodeIf)(nil)

type nodeFor struct {
	clause *nodeGoCode
	block  *nodeBlock
}

func (e nodeFor) Pos() span             { return e.clause.pos }
func (e *nodeFor) accept(v nodeVisitor) { v.visitFor(e) }

// A nodeBlock represents a block of nodes, i.e., a sequence of nodes that
// appear in order in the source syntax.
type nodeBlock struct {
	nodes []node
}

func (e *nodeBlock) Pos() span {
	// FIXME(paulsmith): span end all exprs
	return e.nodes[0].Pos()
}
func (e *nodeBlock) accept(v nodeVisitor) { v.visitStmtBlock(e) }

var _ node = (*nodeBlock)(nil)

// nodeElement represents an HTML element, with a start tag, optional
// attributes, optional children, and an end tag.
type nodeElement struct {
	tag      tag
	children []node
	pos      span
}

func (e nodeElement) Pos() span             { return e.pos }
func (e *nodeElement) accept(v nodeVisitor) { v.visitElement(e) }

var _ node = (*nodeElement)(nil)

type nodeImport struct {
	decl importDecl
	pos  span
}

func (e nodeImport) Pos() span             { return e.pos }
func (e *nodeImport) accept(v nodeVisitor) { v.visitImport(e) }

var _ node = (*nodeImport)(nil)

type nodeLayout struct {
	name string
	pos  span
}

func (e nodeLayout) Pos() span             { return e.pos }
func (e *nodeLayout) accept(v nodeVisitor) { v.visitLayout(e) }

var _ node = (*nodeLayout)(nil)

/* ------------------ end of syntax nodes -------------------------*/

type span struct {
	start int
	end   int
}

func optimize(tree *syntaxTree) *syntaxTree {
	//opt := optimizer{}
	//opt.visitNodes(nodeList(tree.nodes))
	tree.nodes = coalesceLiterals(tree.nodes)
	return tree
}

// TODO(paulsmith): this needs to be fleshed out and wired up correctly,
// currently it is not actually in use.
type optimizer struct{}

func (o *optimizer) visitElement(n *nodeElement) {
	nodeList(n.children).accept(o)
}

func (o *optimizer) visitLiteral(n *nodeLiteral) {
}

func (o *optimizer) visitGoStrExpr(n *nodeGoStrExpr) {
}

func (o *optimizer) visitGoCode(n *nodeGoCode) {
}

func (o *optimizer) visitIf(n *nodeIf) {
	n.then.accept(o)
	n.alt.accept(o)
}

func (o *optimizer) visitFor(n *nodeFor) {
	n.block.accept(o)
}

func (o *optimizer) visitStmtBlock(n *nodeBlock) {
	nodeList(n.nodes).accept(o)
}

func (o *optimizer) visitNodes(n []node) {
	n = coalesceLiterals(n)
	for i := range n {
		n[i].accept(o)
	}
}

func (o *optimizer) visitImport(n *nodeImport) {
}

func (o *optimizer) visitLayout(n *nodeLayout) {
}

var _ nodeVisitor = (*optimizer)(nil)

// coalesceLiterals is an optimization that coalesces consecutive HTML literal
// nodes together by concatenating their strings together in a single node.
func coalesceLiterals(nodes []node) []node {
	//before := len(nodes)
	n := 0
	for range nodes[:len(nodes)-1] {
		this, thisOk := nodes[n].(*nodeLiteral)
		next, nextOk := nodes[n+1].(*nodeLiteral)
		if thisOk && nextOk && len(this.str) < 512 {
			this.str += next.str
			this.pos.end = next.pos.end
			nodes = append(nodes[:n+1], nodes[n+2:]...)
		} else {
			n++
		}
	}
	nodes = nodes[:n+1]
	//log.Printf("SAVED %d NODES", before-len(nodes))
	return nodes
}

type page struct {
	layout     string
	imports    []importDecl
	codeBlocks []string
	nodes      []node
}

func postProcessTree(tree *syntaxTree) (*page, error) {
	// FIXME(paulsmith): recurse down into child nodes
	// FIXME(paulsmith): handle nodeGoCode nodes
	layoutSet := false
	page := new(page)
	page.layout = "default"
	n := 0
	for _, e := range tree.nodes {
		switch e := e.(type) {
		case *nodeImport:
			page.imports = append(page.imports, e.decl)
		case *nodeLayout:
			if layoutSet {
				return nil, fmt.Errorf("layout already set as %q", page.layout)
			}
			if e.name == "!" {
				page.layout = ""
			} else {
				page.layout = e.name
			}
			layoutSet = true
		default:
			tree.nodes[n] = e
			n++
		}
	}
	page.nodes = tree.nodes[:n]
	return page, nil
}

func generateCodeToFile(c codeGenUnit, basename string, outputPath string, strategy compilationStrategy) error {
	code, err := genCode(c, basename, strategy)
	if err != nil {
		return fmt.Errorf("code gen: %w", err)
	}

	if err := os.WriteFile(outputPath, code, 0644); err != nil {
		return fmt.Errorf("writing out generated code to file: %w", err)
	}

	return nil
}

type codeGenUnit interface {
	nodes() []node
	lineNo(span) int
}

type pageCodeGen struct {
	source string
	layout string
	page   *page
	route  string
}

func (c *pageCodeGen) nodes() []node {
	return c.page.nodes
}

func (c *pageCodeGen) lineNo(s span) int {
	return lineCount(c.source[:s.start+1])
}

func lineCount(s string) int {
	return strings.Count(s, "\n") + 1
}

type layoutCodeGen struct {
	source string
	tree   *syntaxTree
}

func (l *layoutCodeGen) nodes() []node {
	return l.tree.nodes
}

func (l *layoutCodeGen) lineNo(s span) int {
	return lineCount(l.source[:s.start+1])
}

type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(p []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	var n int
	n, w.err = w.w.Write(p)
	return n, w.err
}

func newErrWriter(w io.Writer) *errWriter {
	return &errWriter{w: w, err: nil}
}

type codeGenerator struct {
	c        codeGenUnit
	strategy compilationStrategy
	basename string
	imports  map[string]bool
	bodyb    bytes.Buffer
	bodyw    *errWriter
}

func newCodeGenerator(c codeGenUnit, basename string, strategy compilationStrategy) *codeGenerator {
	var g codeGenerator
	g.c = c
	g.strategy = strategy
	g.basename = basename
	g.imports = make(map[string]bool)
	g.bodyw = newErrWriter(&g.bodyb)
	return &g
}

func (g *codeGenerator) used(name string) {
	g.imports[name] = true
}

func (g *codeGenerator) nodeLineNo(e node) {
	g.lineNo(g.c.lineNo(e.Pos()))
}

func (g *codeGenerator) lineNo(n int) {
	g.printf("//line %s:%d\n", g.basename+".pushup", n)
}

func (g *codeGenerator) printf(format string, args ...any) {
	fmt.Fprintf(g.bodyw, format, args...)
}

func (g *codeGenerator) generate() {
	g.visitNodes(g.c.nodes())
}

func (g *codeGenerator) visitLiteral(n *nodeLiteral) {
	g.used("io")
	g.nodeLineNo(n)
	g.printf("io.WriteString(w, %s)\n", strconv.Quote(n.str))
}

func (g *codeGenerator) visitElement(n *nodeElement) {
	g.used("io")
	g.nodeLineNo(n)
	g.printf("io.WriteString(w, %s)\n", strconv.Quote(n.tag.start()))
	nodeList(n.children).accept(g)
	g.printf("io.WriteString(w, %s)\n", strconv.Quote(n.tag.end()))
}

func (g *codeGenerator) visitGoStrExpr(n *nodeGoStrExpr) {
	if g.strategy == compileLayout && n.expr == "contents" {
		// NOTE(paulsmith): this is acting sort of like a coroutine, yielding back to the
		// component that is being rendered with this layout
		g.printf(`if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		`)
		g.printf("yield <- struct{}{}\n")
		g.printf("<-yield\n")
	} else {
		g.used("html/template")
		g.used("fmt")
		g.used("io")
		g.nodeLineNo(n)
		g.printf("{\n")
		g.printf("\tvar __x any = %s\n", n.expr)
		g.printf("\tswitch __val := __x.(type) {\n")
		g.printf("\t\tcase string:\n")
		g.printf("\t\t\tio.WriteString(w, template.HTMLEscapeString(__val))\n")
		g.printf("\t\tcase fmt.Stringer:\n")
		g.printf("\t\t\tio.WriteString(w, template.HTMLEscapeString(__val.String()))\n")
		g.printf("\t\tcase []byte:\n")
		g.printf("\t\t\ttemplate.HTMLEscape(w, __val)\n")
		g.printf("\t\tdefault:\n")
		g.printf("\t\t\tpanic(\"expected a string, []bytes, or fmt.Stringer expression\")\n")
		g.printf("\t}\n")
		g.printf("}\n")
	}
}

func (g *codeGenerator) visitGoCode(n *nodeGoCode) {
	srcLineNo := g.c.lineNo(n.Pos())
	lines := strings.Split(n.code, "\n")
	for _, line := range lines {
		g.lineNo(srcLineNo)
		g.printf("%s\n", line)
		srcLineNo++
	}
}

func (g *codeGenerator) visitIf(n *nodeIf) {
	g.printf("if %s {\n", n.cond.expr)
	n.then.accept(g)
	if n.alt == nil {
		g.printf("}\n")
	} else {
		g.printf("} else {\n")
		n.alt.accept(g)
		g.printf("}\n")
	}
}

func (g *codeGenerator) visitFor(n *nodeFor) {
	g.printf("for %s {\n", n.clause.code)
	n.block.accept(g)
	g.printf("}\n")
}

func (g *codeGenerator) visitStmtBlock(n *nodeBlock) {
	for _, e := range n.nodes {
		e.accept(g)
	}
}

func (g *codeGenerator) visitNodes(n []node) {
	for _, e := range n {
		e.accept(g)
	}
}

func (g *codeGenerator) visitLayout(n *nodeLayout) {
	// no-op
}

func (g *codeGenerator) visitImport(n *nodeImport) {
	// no-op
}

var _ nodeVisitor = (*codeGenerator)(nil)

func genCode(c codeGenUnit, basename string, strategy compilationStrategy) ([]byte, error) {
	var (
		headerb  bytes.Buffer
		importsb bytes.Buffer
	)

	g := newCodeGenerator(c, basename, strategy)

	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	fmt.Fprintf(&headerb, "// this file is mechanically generated, do not edit!\n")
	fmt.Fprintf(&headerb, "package %s\n", packageName)

	typeName := genStructName(basename, strategy)

	type field struct {
		name string
		typ  string
	}

	fields := []field{}

	fmt.Fprintf(g.bodyw, "type %s struct {\n", typeName)
	for _, field := range fields {
		fmt.Fprintf(g.bodyw, "%s %s\n", field.name, field.typ)
	}
	fmt.Fprintf(g.bodyw, "}\n")

	switch strategy {
	case compilePushupPage:
		fmt.Fprintf(g.bodyw, "func (t *%s) register() {\n", typeName)
		fmt.Fprintf(g.bodyw, "routes.add(\"%s\", t)\n", c.(*pageCodeGen).route)
		fmt.Fprintf(g.bodyw, "}\n\n")

		fmt.Fprintf(g.bodyw, "func init() {\n")
		fmt.Fprintf(g.bodyw, "(&%s{}).register()\n", typeName)
		fmt.Fprintf(g.bodyw, "}\n\n")
	case compileLayout:
		fmt.Fprintf(g.bodyw, "func init() {\n")
		fmt.Fprintf(g.bodyw, "layouts[\"%s\"] = &%s{}\n", basename, typeName)
		fmt.Fprintf(g.bodyw, "}\n\n")
	}

	g.used("io")
	g.used("net/http")
	switch strategy {
	case compilePushupPage:
		fmt.Fprintf(g.bodyw, "func (t *%s) Render(w io.Writer, req *http.Request) error {\n", typeName)
	case compileLayout:
		fmt.Fprintf(g.bodyw, "func (t *%s) Render(yield chan struct{}, w io.Writer, req *http.Request) error {\n", typeName)
	default:
		panic("")
	}

	if strategy == compilePushupPage {
		comp := c.(*pageCodeGen)
		if comp.layout != "" {
			// TODO(paulsmith): this is where a flag that could conditionally toggle the rendering
			// of the layout could go - maybe a special header in request object?
			g.used("golang.org/x/sync/errgroup")
			fmt.Fprintf(g.bodyw,
				`g := new(errgroup.Group)
				yield := make(chan struct{})
				layout := getLayout("%s")
				g.Go(func() error {
					if err := layout.Render(yield, w, req); err != nil {
						return err
					}
					return nil
				})
				// Let layout render run until its @contents is encountered
				<-yield
			`, comp.layout)
		}
	}

	// Make a new scope for the user's code block and HTML. This will help (but not fully prevent)
	// name collisions with the surrounding code.
	fmt.Fprintf(g.bodyw, "// Begin user Go code and HTML\n")
	fmt.Fprintf(g.bodyw, "{\n")

	g.generate()

	if strategy == compilePushupPage {
		comp := c.(*pageCodeGen)
		if comp.layout != "" {
			fmt.Fprintf(g.bodyw,
				`yield <- struct{}{}
				if err := g.Wait(); err != nil {
					return err
				}
			`)
		}
	}

	// Close the scope we started for the user code and HTML.
	fmt.Fprintf(g.bodyw, "// End user Go code and HTML\n")
	fmt.Fprintf(g.bodyw, "}\n")

	fmt.Fprintf(g.bodyw, "return nil\n")
	fmt.Fprintf(g.bodyw, "}\n")

	if g.bodyw.err != nil {
		return nil, fmt.Errorf("problem writing to the codegen buffer: %w", g.bodyw.err)
	}

	importsb.WriteString("\nimport (\n")
	for im, ok := range g.imports {
		if ok {
			line := fmt.Sprintf("\"%s\"\n", im)
			importsb.WriteString(line)
		}
	}
	importsb.WriteString(")\n\n")
	if c, ok := c.(*pageCodeGen); ok {
		for _, decl := range c.page.imports {
			importsb.WriteString("import ")
			if decl.pkgName != "" {
				importsb.WriteString(decl.pkgName + " ")
			}
			importsb.WriteString(decl.path + "\n")
		}
	}

	var combinedb bytes.Buffer
	combinedb.ReadFrom(&headerb)
	combinedb.ReadFrom(&importsb)
	combinedb.ReadFrom(&g.bodyb)

	//fmt.Fprintf(os.Stderr, "\x1b[36m%s\x1b[0m", combinedb.String())

	formatted, err := format.Source(combinedb.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt the generated code: %w", err)
	}

	return formatted, nil
}

var structNameIdx int = 0

func safeGoIdentFromFilename(filename string) string {
	// FIXME(paulsmith): need to be more rigorous in mapping safely from
	// filenames to legal Go identifiers
	return strings.ReplaceAll(strings.ReplaceAll(filename, ".", ""), "-", "_")
}

func genStructName(basename string, strategy compilationStrategy) string {
	structNameIdx++
	basename = safeGoIdentFromFilename(basename)
	if strategy == compileLayout {
		basename += "_layout"
	}
	return "Pushup__" + basename + "__" + strconv.Itoa(structNameIdx)
}

type importDecl struct {
	pkgName string
	path    string
}

type syntaxTree struct {
	nodes []node
}

func init() {
	if atom.Lookup([]byte("text")) != 0 {
		panic("expected <text> to not be a common HTML tag")
	}
}

func parse(source string) (*syntaxTree, error) {
	var p parser
	p.src = source
	p.offset = 0
	p.htmlParser = &htmlParser{parser: &p}
	p.codeParser = &codeParser{parser: &p}
	tree := p.htmlParser.parseDocument()
	if len(p.errs) > 0 {
		return nil, p.errs[0]
	}
	return tree, nil
}

type parser struct {
	src        string
	offset     int
	errs       []error
	htmlParser *htmlParser
	codeParser *codeParser
}

func (p *parser) source() string {
	return p.sourceFrom(p.offset)
}

func (p *parser) sourceFrom(offset int) string {
	return p.src[offset:]
}

func (p *parser) errorf(format string, args ...any) {
	p.errs = append(p.errs, fmt.Errorf(format, args...))
	log.Printf("\x1b[0;31mERROR: %v\x1b[0m", p.errs[len(p.errs)-1])
}

type htmlParser struct {
	parser *parser

	// current token
	tok html.Token
	err error
	raw string
	// the global parser offset at the beginning of a new token
	start int
}

func (p *htmlParser) advance() {
	// NOTE(paulsmith): we're re-creating a tokenizer each time through
	// the loop, with the starting point of the source text moved up by the
	// length of the previous token, in order to synchronize the position
	// in the source between the code parser and the HTML parser. this is
	// probably inefficient and could be done "better" and more efficiently
	// by reusing the tokenizer, as for sure it generates more garbage. but
	// would need to profile to see if this is actually a big problem to
	// end users, and in any case, it's only during compilation, so doesn't
	// impact the runtime web application.
	tokenizer := html.NewTokenizer(strings.NewReader(p.parser.source()))
	tokenizer.SetMaxBuf(0) // unlimited buffer size
	tokenizer.Next()
	p.err = tokenizer.Err()
	p.raw = string(tokenizer.Raw())
	p.tok = tokenizer.Token()
	p.start = p.parser.offset
	p.parser.offset += len(p.raw)
}

func isAllWhitespace(s string) bool {
	for s != "" {
		r, size := utf8.DecodeRuneInString(s)
		if !unicode.IsSpace(r) {
			return false
		}
		s = s[size:]
	}
	return true
}

func (p *htmlParser) skipWhitespace() []*nodeLiteral {
	var result []*nodeLiteral
	for {
		if p.tok.Type == html.TextToken && isAllWhitespace(p.raw) {
			n := nodeLiteral{str: p.raw, pos: span{start: p.start, end: p.parser.offset}}
			result = append(result, &n)
			p.advance()
		} else {
			break
		}
	}
	return result
}

func (p *htmlParser) parseDocument() *syntaxTree {
	tree := new(syntaxTree)
tokenLoop:
	for {
		p.advance()
		if p.tok.Type == html.ErrorToken {
			if p.err == io.EOF {
				break tokenLoop
			} else {
				p.parser.errorf("HTML tokenizer: %w", p.err)
			}
		}
		// FIXME(paulsmith): allow @ transition in an attribute
		if idx := strings.IndexRune(p.raw, '@'); idx >= 0 && p.tok.Type != html.StartTagToken {
			if escapedAt := strings.Index(p.raw, "@@"); escapedAt >= 0 {
				// it's an escaped @
				if escapedAt > 0 {
					// emit the leading text before the "@@"
					e := new(nodeLiteral)
					e.pos.start = p.start
					e.pos.end = p.start + escapedAt
					e.str = p.raw[:escapedAt]
					tree.nodes = append(tree.nodes, e)
				}
				e := new(nodeLiteral)
				e.pos.start = p.start + escapedAt
				e.pos.end = p.start + escapedAt + 2
				e.str = "@"
				tree.nodes = append(tree.nodes, e)
				p.parser.offset = p.start + escapedAt + 2
			} else {
				// TODO(paulsmith): check for an email address
				// FIXME(paulsmith): clean this up!
				if strings.HasPrefix(p.raw[idx+1:], "layout") {
					s := p.raw[idx+1+len("layout"):]
					n := 0
					if len(s) < 1 || s[0] != ' ' {
						p.parser.errorf("@layout must be followed by a space")
						break tokenLoop
					}
					s = s[1:]
					n++
					e := new(nodeLayout)
					if len(s) > 0 && s[0] == '!' {
						e.name = "!"
						n++
					} else {
						var name []rune
						for {
							r, size := utf8.DecodeRuneInString(s)
							if r == 0 {
								break
							}
							if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-' || r == '.' {
								name = append(name, r)
								s = s[size:]
								n += size
							} else {
								break
							}
						}
						e.name = string(name)
					}
					e.pos.start = p.start + idx + 1
					newOffset := e.pos.start + len("layout") + n
					e.pos.end = newOffset
					p.parser.offset = newOffset
					tree.nodes = append(tree.nodes, e)
				} else {
					newOffset := p.start + idx + 1
					p.parser.offset = newOffset
					leading := p.raw[:idx]
					if idx > 0 {
						var htmlNode nodeLiteral
						htmlNode.pos.start = p.start
						htmlNode.pos.end = p.start + len(leading)
						htmlNode.str = leading
						tree.nodes = append(tree.nodes, &htmlNode)
					}
					e := p.transition()
					// NOTE(paulsmith): this bubbles up nil due to parseImportKeyword,
					// the result of which we don't treat as a node in the syntax tree
					if e != nil {
						tree.nodes = append(tree.nodes, e)
					}
				}
			}
		} else {
			e := new(nodeLiteral)
			e.pos.start = p.start
			e.pos.end = p.parser.offset
			e.str = p.raw
			tree.nodes = append(tree.nodes, e)
		}
	}
	return tree
}

func (p *htmlParser) transition() node {
	preview := p.parser.source()
	if len(preview) > 40 {
		preview = preview[:40]
	}
	codeParser := p.parser.codeParser
	codeParser.reset()
	e := codeParser.parseCodeBlock()
	return e
}

type tag struct {
	name string
	attr []html.Attribute
}

func (t tag) String() string {
	if len(t.attr) == 0 {
		return t.name
	}
	buf := bytes.NewBufferString(t.name)
	for _, a := range t.attr {
		buf.WriteByte(' ')
		buf.WriteString(a.Key)
		buf.WriteString(`="`)
		buf.WriteString(html.EscapeString(a.Val))
		buf.WriteByte('"')
	}
	return buf.String()
}

func (t tag) start() string {
	return "<" + t.String() + ">"
}

func (t tag) end() string {
	return "</" + t.String() + ">"
}

func tok2tag(tok html.Token) tag {
	return tag{name: tok.Data, attr: tok.Attr}
}

func (p *htmlParser) match(typ html.TokenType) bool {
	return p.tok.Type == typ
}

func (p *htmlParser) parseElement() *nodeElement {
	var result *nodeElement

	if !p.match(html.StartTagToken) {
		p.parser.errorf("expected an HTML element start tag, got %s", p.tok.Type)
		return result
	}

	result = new(nodeElement)
	result.tag = tok2tag(p.tok)
	result.pos.start = p.parser.offset - len(p.raw)
	result.pos.end = p.parser.offset
	p.advance()

	result.children = p.parseChildren()

	if !p.match(html.EndTagToken) {
		p.parser.errorf("expected an HTML element end tag, got %q", p.tok.Type)
		return result
	}

	if result.tag.name != p.tok.Data {
		p.parser.errorf("expected </%s> end tag, got </%s>", result.tag.name, p.tok.Data)
	}

	return result
}

func sprintStartTag(elems []*nodeElement) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "[")
	for i, e := range elems {
		fmt.Fprintf(&buf, "%s", e.tag.name)
		if i < len(elems)-1 {
			fmt.Fprintf(&buf, " ")
		}
	}
	fmt.Fprintf(&buf, "]")
	return buf.String()
}

func sprintNodes(nodes []node) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "[")
	for i, n := range nodes {
		switch n := n.(type) {
		case *nodeLiteral:
			fmt.Fprintf(&buf, "%q", n.str)
		case *nodeElement:
			fmt.Fprintf(&buf, "%s", n.tag.name)
		}
		if i < len(nodes)-1 {
			fmt.Fprintf(&buf, " ")
		}
	}
	fmt.Fprintf(&buf, "]")
	return buf.String()
}

func (p *htmlParser) parseChildren() []node {
	var result []node // either *nodeElement or *nodeLiteral
	var elemStack []*nodeElement
loop:
	for {
		switch p.tok.Type {
		case html.ErrorToken:
			if p.err == io.EOF {
				break loop
			} else {
				p.parser.errorf("HTML tokenizer: %w", p.err)
			}
		// FIXME(paulsmith): handle self-closing tags/elements
		case html.StartTagToken:
			elem := new(nodeElement)
			elem.tag = tok2tag(p.tok)
			elem.pos.start = p.parser.offset - len(p.raw)
			elem.pos.end = p.parser.offset
			p.advance()
			elem.children = p.parseChildren()
			result = append(result, elem)
			elemStack = append(elemStack, elem)
		case html.EndTagToken:
			if len(elemStack) == 0 {
				return result
			}
			elem := elemStack[len(elemStack)-1]
			if elem.tag.name == p.tok.Data {
				elemStack = elemStack[:len(elemStack)-1]
				p.advance()
			} else {
				p.parser.errorf("mismatch end tag, expected </%s>, got </%s>", elem.tag.name, p.tok.Data)
				return result
			}
		case html.TextToken:
			// TODO(paulsmith): de-dupe this logic
			if idx := strings.IndexRune(p.raw, '@'); idx >= 0 {
				if idx < len(p.raw)-1 && p.raw[idx+1] == '@' {
					// it's an escaped @
					// TODO(paulsmith): emit '@' literal text expression
				} else {
					// TODO(paulsmith): check for an email address
					newOffset := p.start + idx + 1
					p.parser.offset = newOffset
					leading := p.raw[:idx]
					if idx > 0 {
						var htmlNode nodeLiteral
						htmlNode.pos.start = p.start
						htmlNode.pos.end = p.start + len(leading)
						htmlNode.str = leading
						result = append(result, &htmlNode)
					}
					e := p.transition()
					result = append(result, e)
				}
			} else {
				var htmlNode nodeLiteral
				htmlNode.pos.start = p.start
				htmlNode.pos.end = p.parser.offset
				htmlNode.str = p.raw
				result = append(result, &htmlNode)
			}
			p.advance()
		case html.CommentToken:
			// ???
		case html.DoctypeToken:
			// ???
		default:
			panic("")
		}
	}

	return result
}

type codeParser struct {
	parser         *parser
	baseOffset     int
	file           *token.File
	scanner        *scanner.Scanner
	acceptedToken  goToken
	lookaheadToken goToken
}

func (p *codeParser) reset() {
	p.baseOffset = p.parser.offset
	fset := token.NewFileSet()
	source := p.parser.source()
	p.file = fset.AddFile("", fset.Base(), len(source))
	p.scanner = new(scanner.Scanner)
	p.scanner.Init(p.file, []byte(source), p.handleGoScanErr, scanner.ScanComments)
	p.acceptedToken = goToken{}
	p.lookaheadToken = goToken{}
}

func (p *codeParser) source() string {
	return p.parser.sourceFrom(p.baseOffset)
}

func (p *codeParser) sourceFrom(pos token.Pos) string {
	return p.parser.sourceFrom(p.baseOffset + p.file.Offset(pos))
}

func (p *codeParser) lookahead() (t goToken) {
	t.pos, t.tok, t.lit = p.scanner.Scan()
	return t
}

func (p *codeParser) handleGoScanErr(pos token.Position, msg string) {
	p.parser.errorf("Go scanning error: pos: %v msg: %s", pos, msg)
}

type goToken struct {
	pos token.Pos
	tok token.Token
	lit string
}

func (t goToken) String() string {
	if t.tok.IsLiteral() || t.tok == token.IDENT {
		return t.lit
	}
	return t.tok.String()
}

func (p *codeParser) peek() goToken {
	if p.lookaheadToken.pos == 0 {
		p.lookaheadToken = p.lookahead()
	}
	return p.lookaheadToken
}

func (p *codeParser) prev() goToken {
	return p.acceptedToken
}

func (p *codeParser) advance() {
	t := p.peek()
	// the Go scanner skips over whitespace so we need to be careful about the
	// logic for advancing the main parser internal source offset.
	p.parser.offset = p.baseOffset + p.file.Offset(t.pos) + len(t.String())
	p.acceptedToken = t
	p.lookaheadToken = p.lookahead()
}

func (p *codeParser) transition() *nodeBlock {
	htmlParser := p.parser.htmlParser
	htmlParser.advance()
	var stmtBlock nodeBlock
	ws := htmlParser.skipWhitespace()
	for _, n := range ws {
		stmtBlock.nodes = append(stmtBlock.nodes, n)
	}
	elem := htmlParser.parseElement()
	stmtBlock.nodes = append(stmtBlock.nodes, elem)
	p.reset()
	return &stmtBlock
}

func (p *codeParser) parseCodeBlock() node {
	// starting at the token just past the '@' indicating a transition from HTML
	// parsing to Go code parsing
	var e node
	if p.peek().tok == token.IF {
		p.advance()
		e = p.parseIfStmt()
	} else if p.peek().tok == token.IDENT && p.peek().lit == "code" {
		p.advance()
		e = p.parseCodeKeyword()
	} else if p.peek().tok == token.IMPORT {
		p.advance()
		e = p.parseImportKeyword()
	} else if p.peek().tok == token.FOR {
		p.advance()
		e = p.parseForStmt()
	} else if p.peek().tok == token.LPAREN {
		p.advance()
		e = p.parseExplicitExpression()
	} else if p.peek().tok == token.IDENT {
		e = p.parseImplicitExpression()
	} else if p.peek().tok == token.EOF {
		p.parser.errorf("unexpected EOF")
	} else {
		panic("unexpected token type: " + p.peek().tok.String())
	}
	return e
}

func (p *codeParser) parseIfStmt() *nodeIf {
	var stmt nodeIf
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.parser.errorf("premature end of conditional in IF statement")
			break loop
		case token.LBRACE:
			// conditional expression has been scanned
			break loop
		// TODO(paulsmith): add cases for tokens that are illegal in an expression
		default:
			p.advance()
		}
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	offset := p.baseOffset + p.file.Offset(start)
	stmt.cond = new(nodeGoStrExpr)
	stmt.cond.pos.start = offset
	stmt.cond.pos.end = offset + n
	stmt.cond.expr = p.sourceFrom(start)[:n]
	if _, err := goparser.ParseExpr(stmt.cond.expr); err != nil {
		p.parser.errorf("parsing Go expression in IF conditional: %w", err)
	}
	stmt.then = p.parseStmtBlock()
	if p.peek().tok == token.ELSE {
		p.advance()
		elseBlock := p.parseStmtBlock()
		stmt.alt = elseBlock
	}
	return &stmt
}

func (p *codeParser) parseForStmt() *nodeFor {
	var stmt nodeFor
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.EOF:
			p.parser.errorf("premature end of clause in FOR statement")
			break loop
		case token.LBRACE:
			break loop
		default:
			p.advance()
		}
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	offset := p.baseOffset + p.file.Offset(start)
	stmt.clause = new(nodeGoCode)
	stmt.clause.pos.start = offset
	stmt.clause.pos.end = offset + n
	stmt.clause.code = p.sourceFrom(start)[:n]
	stmt.block = p.parseStmtBlock()
	return &stmt
}

func (p *codeParser) parseStmtBlock() *nodeBlock {
	// we are sitting on the opening '{' token here
	if p.peek().tok != token.LBRACE {
		panic("")
	}
	p.advance()
	var block *nodeBlock
	// it is likely non-Go code (i.e., HTML, or HTML and a transition)
	switch p.peek().tok {
	case token.ILLEGAL:
		if p.peek().lit == "@" {
			p.scanner.ErrorCount--
			p.advance()
			// we can just stay in the code parser
			codeBlock := p.parseCodeBlock()
			block = &nodeBlock{nodes: []node{codeBlock}}
		}
	case token.EOF:
		p.parser.errorf("premature end of block in IF statement")
	default:
		block = p.transition()
	}
	// we should be at the closing '}' token here
	if p.peek().tok != token.RBRACE {
		p.parser.errorf("expected closing '}', got %v", p.peek())
	}
	p.advance()
	return block
}

func (p *codeParser) parseCodeKeyword() *nodeGoCode {
	var result nodeGoCode
	// we are one token past the 'code' keyword
	if p.peek().tok != token.LBRACE {
		p.parser.errorf("expected '{', got '%s'", p.peek().tok)
	}
	depth := 1
	p.advance()
	result.pos.start = p.parser.offset
	start := p.peek().pos
loop:
	for {
		switch p.peek().tok {
		case token.LBRACE:
			depth++
		case token.RBRACE:
			depth--
			if depth == 0 {
				break loop
			}
		}
		p.advance()
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	if p.peek().tok != token.RBRACE {
		panic("")
	}
	p.advance()
	result.code = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	return &result
}

func (p *codeParser) parseImportKeyword() *nodeImport {
	/*
		examples
		@import   "lib/math"         math.Sin
		@import m "lib/math"         m.Sin
		@import . "lib/math"         Sin
	*/
	e := new(nodeImport)
	// we are one token past the 'code' keyword
	switch p.peek().tok {
	case token.STRING:
		e.decl.path = p.peek().lit
		p.advance()
	case token.IDENT:
		e.decl.pkgName = p.peek().lit
		p.advance()
		if p.peek().tok != token.STRING {
			p.parser.errorf("expected string, got %s", p.peek().tok)
			return e
		}
		e.decl.path = p.peek().lit
	case token.PERIOD:
		e.decl.pkgName = "."
		p.advance()
		if p.peek().tok != token.STRING {
			p.parser.errorf("expected string, got %s", p.peek().tok)
			return e
		}
		e.decl.path = p.peek().lit
	default:
		p.parser.errorf("unexpected token type after @import: %s", p.peek().tok)
	}
	return e
}

func (p *codeParser) parseExplicitExpression() *nodeGoStrExpr {
	// one token past the opening '('
	var result nodeGoStrExpr
	result.pos.start = p.parser.offset
	start := p.peek().pos
	depth := 1
loop:
	for {
		switch p.peek().tok {
		case token.LPAREN:
			depth++
		case token.RPAREN:
			depth--
			if depth == 0 {
				break loop
			}
		default:
		}
		p.advance()
	}
	n := (p.file.Offset(p.prev().pos) - p.file.Offset(start)) + len(p.prev().String())
	if p.peek().tok != token.RPAREN {
		panic("")
	}
	p.advance()
	result.expr = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.parser.errorf("illegal Go expression: %w", err)
	}
	return &result
}

func (p *codeParser) parseImplicitExpression() *nodeGoStrExpr {
	if p.peek().tok != token.IDENT {
		panic("")
	}
	var result nodeGoStrExpr
	result.pos.start = p.parser.offset
	start := p.peek().pos
	n := len(p.peek().String())
	p.advance()
	for {
		if p.peek().tok == token.PERIOD {
			p.advance()
			if p.peek().tok == token.IDENT {
				n += 1 + len(p.peek().String())
				p.advance()
			} else {
				p.parser.errorf("illegal selector expression")
				break
			}
		} else {
			break
		}
	}
	result.expr = p.sourceFrom(start)[:n]
	result.pos.end = result.pos.start + n
	if _, err := goparser.ParseExpr(result.expr); err != nil {
		p.parser.errorf("illegal Go expression: %w", err)
	}
	return &result
}

type debugPrettyPrinter struct {
	w     io.Writer
	depth int
}

var _ nodeVisitor = (*debugPrettyPrinter)(nil)

const pad = "    "

func acceptAndIndent(n node, p *debugPrettyPrinter) {
	p.depth++
	n.accept(p)
	p.depth--
}

func (p *debugPrettyPrinter) visitLiteral(n *nodeLiteral) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[32m%q\x1b[0m", n.str)
	fmt.Fprintln(p.w, "")
}

func (p *debugPrettyPrinter) visitGoStrExpr(n *nodeGoStrExpr) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[33m%s\x1b[0m", n.expr)
	fmt.Fprintln(p.w, "")
}

func (p *debugPrettyPrinter) visitGoCode(n *nodeGoCode) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[34m%s\x1b[0m", n.code)
	fmt.Fprintln(p.w, "")
}

func (p *debugPrettyPrinter) visitIf(n *nodeIf) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[35mIF\x1b[0m\n")
	acceptAndIndent(n.cond, p)
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[35mTHEN\x1b[0m\n")
	acceptAndIndent(n.then, p)
	if n.alt != nil {
		p.w.Write([]byte(strings.Repeat(pad, p.depth)))
		fmt.Fprintf(p.w, "\x1b[1;35mELSE\x1b[0m\n")
		acceptAndIndent(n.alt, p)
	}
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[35mEND IF\x1b[0m\n")
}

func (p *debugPrettyPrinter) visitFor(n *nodeFor) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[36mFOR\x1b[0m\n")
	acceptAndIndent(n.clause, p)
	acceptAndIndent(n.block, p)
}

func (p *debugPrettyPrinter) visitElement(n *nodeElement) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "\x1b[31m%s\x1b[0m\n", n.tag.start())
	for _, e := range n.children {
		acceptAndIndent(e, p)
	}
	fmt.Fprintf(p.w, "\x1b[31m%s\x1b[0m\n", n.tag.end())
}

func (p *debugPrettyPrinter) visitStmtBlock(n *nodeBlock) {
	nodeList(n.nodes).accept(p)
}

func (p *debugPrettyPrinter) visitNodes(nodes []node) {
	for _, n := range nodes {
		acceptAndIndent(n, p)
	}
}

func (p *debugPrettyPrinter) visitImport(n *nodeImport) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "IMPORT ")
	if n.decl.pkgName != "" {
		fmt.Fprintf(p.w, "%s", n.decl.pkgName)
	}
	fmt.Fprintf(p.w, "%s\n", n.decl.path)
}

func (p *debugPrettyPrinter) visitLayout(n *nodeLayout) {
	p.w.Write([]byte(strings.Repeat(pad, p.depth)))
	fmt.Fprintf(p.w, "LAYOUT %s\n", n.name)
}
