package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

var outDir = "./build"

func main() {
	singleFlag := flag.String("single", "", "path to a single Pushup file")
	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")

	flag.Parse()

	var pushupFiles []string

	os.RemoveAll(outDir)

	appDir := "app"
	if flag.NArg() == 1 {
		appDir = flag.Arg(0)
	}
	pagesDir := filepath.Join(appDir, "pages")

	var defaultLayout parseResult

	if *singleFlag != "" {
		pushupFiles = []string{*singleFlag}
	} else {
		var err error
		layoutsDir := filepath.Join(appDir, "layouts")
		defaultLayout, err = compileLayout(filepath.Join(layoutsDir, "default.pushup"))
		if err != nil {
			log.Fatalf("compiling default pushup layout: %v", err)
		}

		if !dirExists(pagesDir) {
			log.Fatalf("invalid Pushup project directory structure: couldn't find `pages` subdir")
		}

		entries, err := os.ReadDir(pagesDir)
		if err != nil {
			log.Fatalf("reading app directory: %v", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pushup") {
				path := filepath.Join(pagesDir, entry.Name())
				// log.Printf("found pushup file: %s", path)
				pushupFiles = append(pushupFiles, path)
			}
		}
	}

	for _, path := range pushupFiles {
		err := compilePushup(defaultLayout, path)
		if err != nil {
			log.Fatalf("compiling pushup file %s: %v", path, err)
		}
	}

	addSupport(outDir)

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

func addSupport(dir string) {
	supportFile := `
package build

import (
	"io"
	"net/http"
	"errors"
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

func (r *routeList) add(pattern string, c component) {
	*r = append(*r, newRoute(pattern, c))
}

type route struct {
	regex *regexp.Regexp
	component component
}

func newRoute(pattern string, c component) route {
	return route{regexp.MustCompile("^" + pattern + "$"), c}
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
)

var logger *log.Logger

func main() {
	// FIXME(paulsmith): detect if connected to terminal for VT100 escapes
	logger = log.New(os.Stderr, "[\x1b[36mPUSHUP\x1b[0m] ", 0)

	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	// FIXME(paulsmith): can't have both port and unixSocket non-empty
	flag.Parse()

	// TODO(paulsmith): allow these middlewares to be configurable on/off
	http.Handle("/", panicRecoveryMiddleware(requestLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
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
		logger.Fatalf("getting a listener: %v", err)
	}

	fmt.Fprintf(os.Stdout, "\x1b[32m↑↑ Pushup ready and listening on %s ↑↑\x1b[0m\n", ln.Addr().String())
	if err := http.Serve(ln, nil); err != nil {
		logger.Fatalf("serving HTTP: %v", err)
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
		} else {
			panic(err)
		}
	}
	return fi.IsDir()
}

func compileLayout(path string) (parseResult, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return parseResult{}, fmt.Errorf("reading pushup layout file: %w", err)
	}

	parsedLayout, err := parsePushup(string(contents))
	if err != nil {
		return parseResult{}, fmt.Errorf("parsing pushup layout file: %w", err)
	}

	return parsedLayout, nil
}

func compilePushup(layout parseResult, path string) error {
	// FIXME(paulsmith): specify output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outDir, err)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading pushup file: %w", err)
	}

	parsedPage, err := parsePushup(string(contents))
	if err != nil {
		return fmt.Errorf("parsing pushup file: %w", err)
	}

	filename := filepath.Base(path)
	basename := strings.TrimSuffix(filename, ".pushup")

	// FIXME(paulsmith): pass in output/build dir path instead of being a package global
	if err := genCode(layout, parsedPage, basename, filepath.Join(outDir, basename+".go")); err != nil {
		return fmt.Errorf("generating Go code from parse result: %w", err)
	}

	return nil
}

type expr interface {
	Pos() span
}

type exprString struct {
	str string
	pos span
}

func (e exprString) Pos() span { return e.pos }

type exprVar struct {
	name string
	pos  span
}

func (e exprVar) Pos() span { return e.pos }

type exprCode struct {
	code string
	pos  span
}

func (e exprCode) Pos() span { return e.pos }

func parsePushup(source string) (parseResult, error) {
	r := strings.NewReader(source)
	t := html.NewTokenizer(r)
	var exprs []expr
	for {
		tt := t.Next()
		if err := t.Err(); errors.Is(err, io.EOF) {
			break
		}
		if tt == html.TextToken {
			text := string(t.Text())
			spans := scanForDirectives(text)
			idx := 0
			for _, s := range spans {
				if s.start > idx {
					exprs = append(exprs, exprString{
						pos: span{start: idx, end: s.start},
						str: text[idx:s.start],
					})
				}
				directive := text[s.start:s.end]
				// log.Printf("@ directive span: %v: %q", s, directive)
				switch {
				case isKeyword(directive[1:]):
					kw := directive[1:]
					switch kw {
					case "code":
						code, end, err := scanCode(text[s.start:], s)
						if err != nil {
							return parseResult{}, fmt.Errorf("scanning code: %w", err)
						}
						exprs = append(exprs, exprCode{
							code: code,
							pos:  span{start: s.start, end: end},
						})
						idx = end + s.start
					default:
						panic("unimplemented keyword " + kw)
					}
				default:
					// variable substitution (technically, expression evaluation)
					exprs = append(exprs, exprVar{
						pos:  s,
						name: directive[1:],
					})
					idx = s.end
				}
			}
			if idx <= len(text)-1 {
				exprs = append(exprs, exprString{
					pos: span{start: idx, end: len(text)},
					str: text[idx:],
				})
			}
		} else {
			exprs = append(exprs, exprString{
				pos: span{},
				str: t.Token().String(),
			})
		}
	}

	/*
		for _, expr := range exprs {
			fmt.Fprintf(os.Stderr, "%T ", expr)
			switch v := expr.(type) {
			case exprString:
				fmt.Fprintf(os.Stderr, "%q\n", v.str)
			case exprVar:
				fmt.Fprintf(os.Stderr, "@%s\n", v.name)
			case exprCode:
				fmt.Fprintf(os.Stderr, "@code {\n%s\n}\n", v.code)
			default:
				panic("unimplemented expr type")
			}
		}
	*/

	// FIXME(paulsmith): don't hardcode layouts
	result := parseResult{layout: "default.pushup", exprs: exprs}
	return result, nil
}

type span struct {
	start int
	end   int
}

func scanForDirectives(text string) []span {
	var spans []span
	idx := 0
	for {
		if idx >= len(text) {
			break
		}
		ch := text[idx]
		if ch == '@' {
			start := idx
			idx++
			for idx < len(text) && isAlphaNumeric(text[idx]) {
				idx++
			}
			s := span{start: start, end: idx}
			spans = append(spans, s)
		} else {
			// no-op
		}
		idx++
	}
	return spans
}

func isAlphaNumeric(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isKeyword(text string) bool {
	return text == "code"
}

type parseResult struct {
	layout string
	exprs  []expr
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

func genCode(layout parseResult, p parseResult, basename string, outputPath string) error {
	var b bytes.Buffer
	w := newErrWriter(&b)

	printf := func(format string, a ...any) {
		fmt.Fprintf(w, format, a...)
	}

	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	printf("// this file is mechanically generated, do not edit!\n")
	printf("package %s\n", packageName)

	imports := []string{"io", "net/http"}

	printf("import (\n")
	for _, import_ := range imports {
		printf("\t\"%s\"\n", import_)
	}
	printf(")\n")

	typeName := genStructName(basename)

	type field struct {
		name string
		typ  string
	}

	fields := []field{}

	printf("type %s struct {\n", typeName)
	for _, field := range fields {
		printf("\t%s %s\n", field.name, field.typ)
	}
	printf("}\n")

	// FIXME(paulsmith): handle nested routes (multiple slashes)
	route := "/" + basename
	if basename == "index" {
		route = "/"
	}

	printf("func (t *%s) register() {\n", typeName)
	printf("\troutes.add(\"%s\", t)\n", route)
	printf("}\n\n")

	printf("func init() {\n")
	printf("\t(&%s{}).register()\n", typeName)
	printf("}\n\n")

	printf("func (t *%s) Render(w io.Writer, req *http.Request) error {\n", typeName)

	// FIXME(paulsmith): what to do about layout @code Go code?
	// first pass over expressions to insert literal Go code at top of the method
	for _, expr := range p.exprs {
		if e, ok := expr.(exprCode); ok {
			printf("%s\n", e.code)
		}
	}

	genCodeForExprs := func(exprs []expr, isLayout bool) []expr {
		for i, expr := range exprs {
			switch v := expr.(type) {
			case exprString:
				printf("\t{\n\t_, err := w.Write([]byte(`%s`))\n", v.str)
				printf("\tif err != nil { return err }\n")
				printf("\t}\n")
			case exprVar:
				if isLayout && v.name == "contents" {
					return exprs[i+1:]
				} else {
					printf("\t{\n\t_, err := w.Write([]byte(%s))\n", v.name)
					printf("\tif err != nil { return err }\n")
					printf("\t}\n")
				}
			case exprCode:
				// no-op
			default:
				panic(fmt.Sprintf("unimplemented expression type %T %v", expr, v))
			}
		}
		return nil
	}

	// for the actual rendered HTML, first generate code for the layout up until the @contents
	// variable is reached, then do this page's, then finish the layout after.
	remainingLayoutExprs := genCodeForExprs(layout.exprs, true)
	_ = genCodeForExprs(p.exprs, false)
	_ = genCodeForExprs(remainingLayoutExprs, true)

	printf("return nil\n")
	printf("}\n")

	if w.err != nil {
		return fmt.Errorf("problem writing to the codegen buffer: %w", w.err)
	}

	// fmt.Printf("\x1b[36m%s\x1b[0m", b.String())

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt the generated code: %w", err)
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("writing out formatted generated code to file: %w", err)
	}

	return nil
}

var structNameIdx int = 0

func genStructName(basename string) string {
	structNameIdx++
	// FIXME(paulsmith): need to be more rigorous in mapping safely from
	// filenames to legal Go type names
	basename = strings.ReplaceAll(strings.ReplaceAll(basename, ".", ""), "-", "_")
	name := "Pushup__" + basename + "__" + strconv.Itoa(structNameIdx)
	return name
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\n' || ch == '\t'
}

func scanCode(text string, s span) (string, int, error) {
	// assert we are sitting on @code at function entry
	if text[:len("@code")] != "@code" {
		panic("assertion error, wanted '@code', got " + text[:len("@code")])
	}
	var start int
	idx := 0
	idx += len("@code")
	for isWhitespace(text[idx]) {
		idx++
	}
	if text[idx] != '{' {
		return "", 0, fmt.Errorf("expected '{', got '%c'", text[idx])
	}
	idx++
	start = idx
	for isWhitespace(text[idx]) {
		idx++
	}
	depth := 1
	for {
		if depth == 0 || idx >= len(text) {
			break
		}
		ch := text[idx]
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
		idx++
	}
	end := idx - 1
	for idx < len(text) && isWhitespace(text[idx]) {
		idx++
	}
	return text[start:end], idx, nil
}
