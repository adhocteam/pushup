package lexer

import (
	"fmt"
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
	// HTML_TEXT............."\n}\n"..................74.....3
	// EOF..................."".......................77.....0
}

func TestMatchesBlockOpen(t *testing.T) {
	tests := []struct {
		in           []byte
		wantMatchLen int
		wantOk       bool
	}{
		{
			[]byte(""),
			0,
			false,
		},
		{
			[]byte("{"),
			0,
			false,
		},
		{
			[]byte("{\n"),
			2,
			true,
		},
		{
			[]byte("{ \n"),
			3,
			true,
		},
		{
			[]byte(" { \n"),
			4,
			true,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			gotMatchLen, gotOk := matchesBlockOpen(test.in)
			if test.wantMatchLen != gotMatchLen {
				t.Errorf("want match len: %d, got: %d", test.wantMatchLen, gotMatchLen)
			}
			if test.wantOk != gotOk {
				t.Errorf("want ok: %t, got: %t", test.wantOk, gotOk)
			}
		})
	}
}
