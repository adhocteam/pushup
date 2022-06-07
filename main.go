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
	"text/template"

	"golang.org/x/net/html"
)

var outDir = "./build"

var singleFlag = flag.String("single", "", "path to a single Pushup file")

func main() {
	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	compileOnly := flag.Bool("compile-only", false, "compile only, don't start web server after")

	flag.Parse()

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
		layoutsDir := filepath.Join(appDir, "layouts")
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
					// log.Printf("found pushup file: %s", path)
					layoutFiles = append(layoutFiles, path)
				}
			}
		}

		pagesDir := filepath.Join(appDir, "pages")
		{
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
	}

	for _, path := range layoutFiles {
		err := compilePushup(path, compileLayout, outDir)
		if err != nil {
			log.Fatalf("compiling layout file %s: %v", path, err)
		}
	}

	for _, path := range pushupFiles {
		err := compilePushup(path, compilePushupComponent, outDir)
		if err != nil {
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

type compilationStrategy int

const (
	compilePushupComponent compilationStrategy = iota
	compileLayout
)

func compilePushup(sourcePath string, strategy compilationStrategy, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", targetDir, err)
	}

	contents, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	parse, err := parsePushup(string(contents))
	if err != nil {
		return fmt.Errorf("parsing file: %w", err)
	}

	var c codeGenUnit
	switch strategy {
	case compilePushupComponent:
		// FIXME(paulsmith): allow user to override layout
		layoutName := "default"
		if *singleFlag != "" {
			layoutName = ""
		}
		c = &componentCodeGen{layout: layoutName, parse: parse}
	case compileLayout:
		c = &layoutCodeGen{parse: parse}
	default:
		panic("")
	}

	filename := filepath.Base(sourcePath)
	basename := strings.TrimSuffix(filename, ".pushup")

	var outputFilename string
	switch strategy {
	case compilePushupComponent:
		outputFilename = basename + ".go"
	case compileLayout:
		outputFilename = basename + "_layout.go"
	default:
		panic("")
	}
	outputPath := filepath.Join(targetDir, outputFilename)

	if err := generateCodeToFile(c, basename, outputPath, strategy); err != nil {
		return fmt.Errorf("generating Go code from parse result: %w", err)
	}

	return nil
}

type expr interface {
	Pos() span
}

type literalType int

const (
	literalHTML literalType = iota
	literalRawString
)

type exprLiteral struct {
	str string
	typ literalType
	pos span
}

func (e exprLiteral) Pos() span { return e.pos }

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
	z := html.NewTokenizer(r)

	const (
		stateStart int = iota
		stateInCode
	)
	state := stateStart
	var accum string
	var start int
	var pos int

	var exprs []expr

	emitCode := func(text string) {
		text = strings.TrimSpace(text)
		text = html.UnescapeString(text) // TODO(paulsmith): check this doesn't break Go code
		text = strings.TrimPrefix(text, "@code {")
		text = strings.TrimSuffix(text, "}")
		exprs = append(exprs, exprCode{
			code: text,
			pos:  span{start: start, end: pos},
		})
	}

loop:
	for {
		tt := z.Next()
		t := z.Token()
		raw := z.Raw()
		pos += len(raw)

		if tt == html.ErrorToken {
			err := z.Err()
			if err == io.EOF {
				break loop
			}
			return parseResult{}, err
		}

		switch state {
		case stateStart:
			if tt == html.TextToken {
				if idx := strings.Index(t.Data, "@code {"); idx != -1 {
					if codeBlockClosed(t.Data) {
						emitCode(t.Data)
						start = pos
					} else {
						state = stateInCode
						accum = t.Data
					}
				} else {
					spans := scanForDirectives(t.Data)
					idx := 0
					for _, s := range spans {
						if s.start > idx {
							exprs = append(exprs, exprLiteral{
								str: t.Data[idx:s.start],
								typ: literalRawString,
								pos: span{start: start + idx, end: start + s.start},
							})
						}
						directive := t.Data[s.start:s.end]
						// log.Printf("@ directive span: %v: %q", s, directive)
						switch {
						case isKeyword(directive[1:]):
							kw := directive[1:]
							switch kw {
							default:
								panic("unimplemented keyword " + kw)
							}
						default:
							// variable substitution (technically, expression evaluation)
							exprs = append(exprs, exprVar{
								pos:  span{start: start + s.start, end: start + s.end},
								name: directive[1:],
							})
							idx = s.end
						}
					}
					if idx <= len(t.Data)-1 {
						exprs = append(exprs, exprLiteral{
							str: t.Data[idx:],
							typ: literalRawString,
							pos: span{start: start + idx, end: start + len(t.Data)},
						})
					}
					start = pos
				}
			} else {
				exprs = append(exprs, exprLiteral{
					str: string(raw),
					typ: literalHTML,
					pos: span{start: start, end: pos},
				})
				start = pos
			}
		case stateInCode:
			accum += string(raw)
			if codeBlockClosed(accum) {
				emitCode(accum)
				state = stateStart
				accum = ""
				start = pos
			}
		}
	}

	/*
		for _, expr := range exprs {
			fmt.Fprintf(os.Stderr, "%T ", expr)
			switch v := expr.(type) {
			case exprLiteral:
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

	// TODO(paulsmith): run @code blocks through the Go parser to catch
	// parse errors at this step and possibly also do some light analysis

	// FIXME(paulsmith): don't hardcode layouts
	result := parseResult{exprs: exprs}

	return result, nil
}

func codeBlockClosed(text string) bool {
	return strings.Count(text, "{") == strings.Count(text, "}")
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
	return false
}

type parseResult struct {
	exprs []expr
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
	exprs() []expr
}

type componentCodeGen struct {
	layout string
	parse  parseResult
}

func (c *componentCodeGen) exprs() []expr {
	return c.parse.exprs
}

type layoutCodeGen struct {
	parse parseResult
}

func (l *layoutCodeGen) exprs() []expr {
	return l.parse.exprs
}

func genCode(c codeGenUnit, basename string, strategy compilationStrategy) ([]byte, error) {
	var (
		headerb  bytes.Buffer
		importsb bytes.Buffer
		bodyb    bytes.Buffer
	)

	bodyw := newErrWriter(&bodyb)

	fprintf := func(w io.Writer, format string, a ...any) {
		fmt.Fprintf(w, format, a...)
	}

	// FIXME(paulsmith): need way to specify this as user
	packageName := "build"

	fprintf(&headerb, "// this file is mechanically generated, do not edit!\n")
	fprintf(&headerb, "package %s\n", packageName)

	imports := map[string]bool{
		"io":                         false,
		"net/http":                   false,
		"html/template":              false,
		"golang.org/x/sync/errgroup": false,
	}
	used := func(name string) {
		imports[name] = true
	}

	typeName := genStructName(basename, strategy)

	type field struct {
		name string
		typ  string
	}

	fields := []field{}

	fprintf(bodyw, "type %s struct {\n", typeName)
	for _, field := range fields {
		fprintf(bodyw, "\t%s %s\n", field.name, field.typ)
	}
	fprintf(bodyw, "}\n")

	switch strategy {
	case compilePushupComponent:
		// FIXME(paulsmith): handle nested routes (multiple slashes)
		route := "/" + basename
		if basename == "index" {
			route = "/"
		}

		fprintf(bodyw, "func (t *%s) register() {\n", typeName)
		fprintf(bodyw, "\troutes.add(\"%s\", t)\n", route)
		fprintf(bodyw, "}\n\n")

		fprintf(bodyw, "func init() {\n")
		fprintf(bodyw, "\t(&%s{}).register()\n", typeName)
		fprintf(bodyw, "}\n\n")
	case compileLayout:
		fprintf(bodyw, "func init() {\n")
		fprintf(bodyw, "\tlayouts[\"%s\"] = &%s{}\n", basename, typeName)
		fprintf(bodyw, "}\n\n")
	}

	used("io")
	used("net/http")
	switch strategy {
	case compilePushupComponent:
		fprintf(bodyw, "func (t *%s) Render(w io.Writer, req *http.Request) error {\n", typeName)
	case compileLayout:
		fprintf(bodyw, "func (t *%s) Render(yield chan struct{}, w io.Writer, req *http.Request) error {\n", typeName)
	default:
		panic("")
	}

	if strategy == compilePushupComponent {
		comp := c.(*componentCodeGen)
		if comp.layout != "" {
			// TODO(paulsmith): this is where a flag that could conditionally toggle the rendering
			// of the layout could go - maybe a special header in request object?
			used("golang.org/x/sync/errgroup")
			fprintf(bodyw, `
	g := new(errgroup.Group)
	yield := make(chan struct{})
	{
			layout := getLayout("%s")
			g.Go(func() error {
				if err := layout.Render(yield, w, req); err != nil {
					return err
				}
				return nil
			})
			<-yield
	}
			`, comp.layout)
		}
	}

	// FIXME(paulsmith): what to do about layout @code Go code?
	// first pass over expressions to insert literal Go code at top of the method
	exprs := c.exprs()
	for _, expr := range exprs {
		if e, ok := expr.(exprCode); ok {
			fprintf(bodyw, "%s\n", e.code)
		}
	}

	for _, expr := range exprs {
		switch v := expr.(type) {
		case exprLiteral:
			switch v.typ {
			case literalHTML:
				used("io")
				fprintf(bodyw, "\t\tio.WriteString(w, %s)\n", strconv.Quote(v.str))
			case literalRawString:
				used("io")
				fprintf(bodyw, "\t\tio.WriteString(w, %s)\n", strconv.Quote(template.HTMLEscapeString(v.str)))
			default:
				panic("unimplemented literal type")
			}
		case exprVar:
			if strategy == compileLayout && v.name == "contents" {
				// NOTE(paulsmith): this is acting sort of like a coroutine, yielding back to the
				// component that is being rendered with this layout
				fprintf(bodyw, "\t\tyield <- struct{}{}\n")
				fprintf(bodyw, "\t\t<-yield\n")
			} else {
				used("html/template")
				// TODO(paulsmith): enforce Stringer() interface on these types
				fprintf(bodyw, "\t\ttemplate.HTMLEscape(w, []byte(%s))\n", v.name)
			}
		case exprCode:
			// no-op
		default:
			panic(fmt.Sprintf("unimplemented expression type %T %v", expr, v))
		}
	}

	if strategy == compilePushupComponent {
		comp := c.(*componentCodeGen)
		if comp.layout != "" {
			fprintf(bodyw, `
	yield <- struct{}{}
	if err := g.Wait(); err != nil {
		return err
	}
`)
		}
	}

	fprintf(bodyw, "\treturn nil\n")
	fprintf(bodyw, "}\n")

	if bodyw.err != nil {
		return nil, fmt.Errorf("problem writing to the codegen buffer: %w", bodyw.err)
	}

	importsb.WriteString("\nimport (\n")
	for import_, ok := range imports {
		if ok {
			line := fmt.Sprintf("\t\"%s\"\n", import_)
			importsb.WriteString(line)
		}
	}
	importsb.WriteString(")\n\n")

	var combinedb bytes.Buffer
	combinedb.ReadFrom(&headerb)
	combinedb.ReadFrom(&importsb)
	combinedb.ReadFrom(&bodyb)

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
