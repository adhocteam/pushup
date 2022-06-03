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

	if *singleFlag != "" {
		pushupFiles = []string{*singleFlag}
	} else {
		appDir := "app"
		if flag.NArg() == 1 {
			appDir = flag.Arg(0)
		}
		pagesDir := filepath.Join(appDir, "pages")
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
				log.Printf("found pushup file: %s", path)
				pushupFiles = append(pushupFiles, path)
			}
		}
	}

	for _, path := range pushupFiles {
		err := compilePushup(path)
		if err != nil {
			log.Fatalf("compiling pushup file %s: %v", path, err)
		}
	}

	var args []string
	if *unixSocket != "" {
		args = []string{"-unix-socket", *unixSocket}
	} else {
		args = []string{"-port", *port}
	}
	if err := buildAndRun(outDir, args); err != nil {
		log.Fatalf("building and running generated Go code: %v", err)
	}
}

func buildAndRun(dir string, passthruArgs []string) error {
	mainExeDir := filepath.Join(dir, "cmd", "myproject")
	if err := os.MkdirAll(mainExeDir, 0755); err != nil {
		return fmt.Errorf("making directory for command: %w", err)
	}

	mainProgram := `package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/AdHocRandD/pushup/build"
)

func main() {
	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	// FIXME(paulsmith): can't have both port and unixSocket non-empty
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		route := &build.PushupIndex1{}
		w.Header().Set("Content-Type", "text/html")
		if err := route.Render(w); err != nil {
			log.Printf("rendering route: %v", err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	})
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
		log.Fatalf("getting a listener: %v", err)
	}
	fmt.Fprintf(os.Stdout, "\x1b[32m↑↑ Pushup ready and listening on %s ↑↑\x1b[0m\n", ln.Addr().String())
	if err := http.Serve(ln, nil); err != nil {
		log.Fatalf("serving HTTP: %v", err)
	}
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

func compilePushup(path string) error {
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
	nameWithoutExt := strings.TrimSuffix(filename, ".pushup")

	if err := genCode(parsedPage, filepath.Join(outDir, nameWithoutExt+".go")); err != nil {
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

	return parseResult{exprs: exprs}, nil
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
	exprs []expr
}

func genCode(p parseResult, outputPath string) error {
	var b bytes.Buffer

	packageName := "build"

	fmt.Fprintf(&b, "// this file is mechnically generated, do not edit!\n")
	fmt.Fprintf(&b, "package %s\n", packageName)

	imports := []string{"io"}

	fmt.Fprintf(&b, "import (\n")
	for _, import_ := range imports {
		fmt.Fprintf(&b, "\t\"%s\"\n", import_)
	}
	fmt.Fprintf(&b, ")\n")

	typeName := "PushupIndex1"

	type field struct {
		name string
		typ  string
	}

	fields := []field{}

	fmt.Fprintf(&b, "type %s struct {\n", typeName)
	for _, field := range fields {
		fmt.Fprintf(&b, "\t%s %s\n", field.name, field.typ)
	}
	fmt.Fprintf(&b, "}\n")

	fmt.Fprintf(&b, "func (t *%s) Render(w io.Writer) error {\n", typeName)

	// first pass over expressions to insert literal Go code at top of the method
	for _, expr := range p.exprs {
		if e, ok := expr.(exprCode); ok {
			fmt.Fprintf(&b, "%s\n", e.code)
		}
	}

	// second pass over all other no-code expressions
	for _, expr := range p.exprs {
		switch v := expr.(type) {
		case exprString:
			fmt.Fprintf(&b, "\t{\n\t_, err := w.Write([]byte(`%s`))\n", v.str)
			fmt.Fprintf(&b, "\tif err != nil { return err }\n")
			fmt.Fprintf(&b, "\t}\n")
		case exprVar:
			fmt.Fprintf(&b, "\t{\n\t_, err := w.Write([]byte(%s))\n", v.name)
			fmt.Fprintf(&b, "\tif err != nil { return err }\n")
			fmt.Fprintf(&b, "\t}\n")
		case exprCode:
			// no-op
		default:
			panic(fmt.Sprintf("unimplemented expression type %T %v", expr, v))
		}
	}
	fmt.Fprintf(&b, "return nil\n")
	fmt.Fprintf(&b, "}\n")

	//fmt.Printf("\x1b[36m%s\x1b[0m", b.String())

	formatted, err := format.Source(b.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt the generated code: %w", err)
	}

	if err := os.WriteFile(outputPath, formatted, 0644); err != nil {
		return fmt.Errorf("writing out formatted generated code to file: %w", err)
	}

	return nil
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
