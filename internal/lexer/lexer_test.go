package lexer

import (
	"fmt"
	"go/scanner"
	"go/token"
	"os"
	"strconv"
	"testing"
	"text/tabwriter"
)

func ExampleLexer() {
	src := []byte(`^if x > 0 {
    <p style="color: red" class=^class>
        hello
    </p>
}
`)
	l := New(src)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, '.', 0)
	fmt.Fprintf(w, "token type\tliteral\tstart\tlen\n")
	for token := range l.Scan() {
		fmt.Fprint(w, token.tokType)
		fmt.Fprint(w, "\t")
		fmt.Fprint(w, strconv.Quote(string(token.literal)))
		fmt.Fprint(w, "\t")
		fmt.Fprint(w, token.Start())
		fmt.Fprint(w, "\t")
		fmt.Fprintln(w, token.Len())
	}
	w.Flush()
	// Output:
	// token type............literal..................start..len
	// ^if..................."if".....................1......2
	// GO_EXPR..............." x > 0 "................3......7
	// HTML_TEXT............."\n    ".................11.....5
	// HTML_START_TAG_NAME..."<p".....................16.....2
	// HTML_ATTR_NAME_TEXT..."style"..................19.....5
	// HTML_ATTR_VALUE_TEXT.."color: red".............26.....10
	// HTML_ATTR_NAME_TEXT..."class"..................38.....5
	// HTML_ATTR_VALUE_GO...."class"..................45.....5
	// HTML_GT...............">"......................50.....1
	// HTML_TEXT............."\n        hello\n    "..51.....19
	// HTML_END_TAG.........."</p>"...................70.....4
	// HTML_TEXT............."\n".....................74.....1
	// GO_BLOCK_CLOSE........"}"......................75.....1
	// HTML_TEXT............."\n".....................76.....1
	// EOF..................."".......................77.....0
}

func TestMatchesBlockOpen(t *testing.T) {
	tests := []struct {
		in           []byte
		wantBracePos int
	}{
		{
			[]byte(""),
			-1,
		},
		{
			[]byte("{"),
			-1,
		},
		{
			[]byte("{\n"),
			0,
		},
		{
			[]byte("{ \n"),
			0,
		},
		{
			[]byte(" { \n"),
			1,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			gotBracePos := matchesBlockOpen(test.in)
			if test.wantBracePos != gotBracePos {
				t.Errorf("want match len: %d, got: %d", test.wantBracePos, gotBracePos)
			}
		})
	}
}

func TestMatchesBlockClose(t *testing.T) {
	tests := []struct {
		in           []byte
		wantBracePos int
	}{
		{
			[]byte(""),
			-1,
		},
		{
			[]byte("}"),
			0,
		},
		{
			[]byte("\n}"),
			1,
		},
		{
			[]byte(" } "),
			1,
		},
		{
			[]byte("\n\t}"),
			2,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			gotBracePos := matchesBlockClose(test.in)
			if test.wantBracePos != gotBracePos {
				t.Errorf("want match len: %d, got: %d", test.wantBracePos, gotBracePos)
			}
		})
	}
}
func xTestGoScanner(_ *testing.T) {
	src := `for i := range 10 {
  println(i)
}`
	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(src))
	var s scanner.Scanner
	s.Init(file, []byte(src), nil, scanner.ScanComments)
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		fmt.Println(pos, tok, lit)
	}
}
