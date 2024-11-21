package lexer

import (
	"fmt"
	"go/token"
	"os"
	"strconv"
	"testing"
	"text/tabwriter"

	"github.com/google/go-cmp/cmp"
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
	// GO_BLOCK_OPEN........."{"......................10.....1
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

func TestKeywords(t *testing.T) {
	bs := newBufGoScanner([]byte(`
        ^for i := range 3 {
            x := i * i
            <p>^i</p>
        }
`), 1)
	for {
		pos, tok, lit := bs.Scan()
		if lit == "" {
			lit = tok.String()
		}
		fmt.Printf("%-2d %10s %10q\n", pos, tok, lit)
		if tok == token.EOF {
			break
		}
	}
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

func TestBufGoScanner(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		sequence func(*bufGoScanner) error
	}{
		{
			"scan consumes token",
			"a = b",
			func(s *bufGoScanner) error {
				p1, t1, _ := s.Scan()
				p2, t2, _ := s.Scan()
				if p1 == p2 && t1 == t2 {
					return fmt.Errorf("successive scan() return same token: %#v", t1)
				}
				return nil
			},
		},
		{
			"unscan after scan",
			"a = b",
			func(s *bufGoScanner) error {
				p1, t1, _ := s.Scan()
				s.Unscan()
				p2, t2, _ := s.Scan()
				if p1 != p2 || t1 != t2 {
					return fmt.Errorf("token after unscan() differs: %#v vs. %#v", t1, t2)
				}
				return nil
			},
		},
		{
			"double unscan panics",
			"a = b",
			func(s *bufGoScanner) (err error) {
				defer func() {
					if r := recover(); r == nil {
						err = fmt.Errorf("unscan() without scan() did not panic")
					}
				}()

				s.Scan()
				s.Unscan()
				// should panic
				s.Unscan()

				return
			},
		},
		{
			"unscan without scan",
			"a = b",
			func(s *bufGoScanner) (err error) {
				defer func() {
					if r := recover(); r == nil {
						err = fmt.Errorf("unscan() without scan() did not panic")
					}
				}()

				// should panic
				s.Unscan()

				return
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			bs := newBufGoScanner([]byte(test.in), 1)
			if err := test.sequence(bs); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestBufHTMLTokenizer(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		sequence func(*bufHTMLTokenizer) error
	}{
		{
			"get consumes token",
			"<p>Foo</p>",
			func(z *bufHTMLTokenizer) error {
				t1 := z.get()
				t2 := z.get()
				if diff := cmp.Diff(t1, t2, cmp.AllowUnexported(htmlToken{})); diff == "" {
					return fmt.Errorf("successive get() return same token: %#v", t1)
				}
				return nil
			},
		},
		{
			"unget after get",
			"<p>Foo</p>",
			func(z *bufHTMLTokenizer) error {
				t1 := z.get()
				z.unget()
				t2 := z.get()
				if diff := cmp.Diff(t1, t2, cmp.AllowUnexported(htmlToken{})); diff != "" {
					return fmt.Errorf("token after unget() differs: %s", diff)
				}
				return nil
			},
		},
		{
			"double unget panics",
			"<p>Foo</p>",
			func(z *bufHTMLTokenizer) (err error) {
				defer func() {
					if r := recover(); r == nil {
						err = fmt.Errorf("unget() without get() did not panic")
					}
				}()
				z.get()
				z.unget()
				// should panic
				z.unget()
				return
			},
		},
		{
			"unget without get",
			"<p>Foo</p>",
			func(z *bufHTMLTokenizer) (err error) {
				defer func() {
					if r := recover(); r == nil {
						err = fmt.Errorf("unget() without get() did not panic")
					}
				}()
				// should panic
				z.unget()
				return
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			bz := newBufHTMLTokenizer([]byte(test.in))
			if err := test.sequence(bz); err != nil {
				t.Error(err)
			}
		})
	}
}
