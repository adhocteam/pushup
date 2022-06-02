package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		log.Fatalf("need path to app directory")
	}

	appDir := flag.Arg(0)

	entries, err := os.ReadDir(appDir)
	if err != nil {
		log.Fatalf("reading app directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pushup") {
			path := filepath.Join(appDir, entry.Name())
			log.Printf("found pushup file: %s", path)
			err := compilePushup(path)
			if err != nil {
				log.Fatalf("compiling pushup file %s: %v", entry.Name(), err)
			}
		}
	}
}

func compilePushup(path string) error {
	// FIXME(paulsmith): specify output directory
	outDir := "./build"

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
						// TODO(paulsmith): insert literal Go code into a function/top-level package/method
					default:
						panic("unimplemented keyword " + kw)
					}
				default:
					// variable substitution (technically, expression evaluation)
					exprs = append(exprs, exprVar{
						pos:  s,
						name: directive[1:],
					})
				}
				idx = s.end
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
		fmt.Fprintf(os.Stderr, "%T %v\n", expr, expr)
	}
	return parseResult{}, nil
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
}

func genCode(p parseResult, outputPath string) error {
	return nil
}
