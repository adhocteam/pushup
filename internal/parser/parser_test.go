package parser

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/source"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var update = flag.Bool("update", false, "update golden files")

func TestParser(t *testing.T) {
	testCases, err := filepath.Glob("testdata/*.up")
	if err != nil {
		t.Fatal(err)
	}

	opts := []cmp.Option{
		cmpopts.EquateEmpty(), // Treats nil slices and empty slices as equal
	}

	for _, inputFile := range testCases {
		t.Run(filepath.Base(inputFile), func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(inputFile)
			if err != nil {
				t.Fatalf("failed to read input file: %v", err)
			}

			parser := New()
			actual, err := parser.Parse(input)
			if err != nil {
				t.Fatalf("unexpected error parsing input: %v", err)
			}

			goldenFile := inputFile[:len(inputFile)-len(".up")] + ".gob"

			if *update || !fileExists(goldenFile) {
				if err := writeDocGolden(actual, goldenFile); err != nil {
					t.Fatalf("failed to create golden file: %v", err)
				}
			} else {
				expected, err := readDocGolden(goldenFile)
				if err != nil {
					t.Fatalf("failed to read golden file: %v", err)
				}

				if diff := cmp.Diff(expected, actual, opts...); diff != "" {
					t.Errorf("unexpected parse result (-expected +actual):\n%s", diff)
				}
			}
		})
	}
}

func writeDocGolden(doc *ast.Document, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	encoder := gob.NewEncoder(f)
	return encoder.Encode(doc)
}

func readDocGolden(filename string) (*ast.Document, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var doc ast.Document
	decoder := gob.NewDecoder(f)
	if err := decoder.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decoding golden file: %w", err)
	}
	return &doc, nil
}

func TestParseSyntaxErrors(t *testing.T) {
	tests := []struct {
		input string
		// expected error conditions
		lineNo int
		column int
	}{
		{"^if", 1, 4},
		// FIXME(paulsmith): add more syntax errors
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			parser := New()
			tree, err := parser.Parse([]byte(tt.input))
			if tree != nil {
				t.Errorf("expected nil tree, got %v", tree)
			}
			if err == nil {
				t.Errorf("expected parse error, got nil")
			}
			serr, ok := err.(syntaxError)
			if !ok {
				t.Errorf("expected syntax error type, got %T", err)
			}
			if tt.lineNo != serr.lineNo || tt.column != serr.column {
				t.Logf("syntax error: %v", serr.err)
				t.Errorf("line:column: want %d:%d, got %d:%d", tt.lineNo, tt.column, serr.lineNo, serr.column)
			}
		})
	}
}

func FuzzParser(f *testing.F) {
	seeds := []string{
		"",
		"<h1>Hello, world!</h1>",
		"^{ name := \"world\" }\n<h1>Hello, ^name!</h1>\n",
		"^if true {\n<a href=\"^req.URL.Path\">this page</a>\n}\n",
		"<div>^(3 + 4 * 5)</div>\n",
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, in []byte) {
		parser := New()
		_, err := parser.Parse(in)
		if err != nil {
			if _, ok := err.(syntaxError); !ok {
				t.Errorf("expected syntax error, got %T %v", err, err)
			}
		}
	})
}

func TestTagString(t *testing.T) {
	tests := []struct {
		tag  ast.Tag
		want string
	}{
		{
			ast.Tag{Name: "h1"},
			"h1",
		},
		{
			ast.Tag{Name: "div", Attrs: []*ast.Attr{{Name: source.StringPos{Text: "class"}, Value: source.StringPos{Text: "banner"}}}},
			"div class=\"banner\"",
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			t.Parallel()
			got := test.tag.String()
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func TestVoidElements(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantVoid bool
	}{
		{
			name:     "br element",
			input:    "<br>",
			wantVoid: true,
		},
		{
			name:     "img element",
			input:    "<img src=\"test.jpg\">",
			wantVoid: true,
		},
		{
			name:     "div element",
			input:    "<div></div>",
			wantVoid: false,
		},
		{
			name:     "input element",
			input:    "<input type=\"text\">",
			wantVoid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()

			doc, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			elem, ok := doc.Nodes.Nodes[0].(*ast.NodeElement)
			if !ok {
				t.Fatal("Expected NodeElement")
			}

			isVoid := elem.Flags&ast.ElementFlagVoid != 0
			if isVoid != tt.wantVoid {
				t.Errorf("got void=%v, want void=%v", isVoid, tt.wantVoid)
			}

			needsClosing := elem.NeedsClosingTag()
			if needsClosing == tt.wantVoid {
				t.Errorf("NeedsClosingTag()=%v, want opposite of void=%v", needsClosing, tt.wantVoid)
			}
		})
	}
}

func TestSelfClosingElements(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantSelfClose bool
	}{
		{
			name:          "self-closing div",
			input:         "<div/>",
			wantSelfClose: true,
		},
		{
			name:          "normal div",
			input:         "<div></div>",
			wantSelfClose: false,
		},
		{
			name:          "self-closing custom element",
			input:         "<my-component/>",
			wantSelfClose: true,
		},
		{
			name:          "self-closing with attributes",
			input:         "<div class=\"test\"/>",
			wantSelfClose: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()

			doc, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			elem, ok := doc.Nodes.Nodes[0].(*ast.NodeElement)
			if !ok {
				t.Fatal("Expected NodeElement")
			}

			isSelfClosing := elem.Flags&ast.ElementFlagSelfClosing != 0
			if isSelfClosing != tt.wantSelfClose {
				t.Errorf("got self-closing=%v, want self-closing=%v", isSelfClosing, tt.wantSelfClose)
			}

			needsClosing := elem.NeedsClosingTag()
			if needsClosing == tt.wantSelfClose {
				t.Errorf("NeedsClosingTag()=%v, want opposite of self-closing=%v", needsClosing, tt.wantSelfClose)
			}
		})
	}
}

func TestMixedElements(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantChildren bool
	}{
		{
			name: "void element inside div",
			input: `<div>
                                <br>
                                <img src="test.jpg">
                        </div>`,
			wantChildren: true,
		},
		{
			name: "self-closing inside div",
			input: `<div>
                                <custom-elem/>
                        </div>`,
			wantChildren: true,
		},
		{
			name:         "multiple void elements",
			input:        "<hr><br><img src=\"test.jpg\">",
			wantChildren: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := New()

			doc, err := p.Parse([]byte(tt.input))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// For inputs with a wrapper div
			if tt.wantChildren {
				elem, ok := doc.Nodes.Nodes[0].(*ast.NodeElement)
				if !ok {
					t.Fatal("Expected NodeElement")
				}

				if elem.Children == nil || elem.Children.Len() == 0 {
					t.Error("Expected children nodes but got none")
				}
			} else {
				// For multiple top-level elements
				if len(doc.Nodes.Nodes) < 2 {
					t.Error("Expected multiple top-level nodes")
				}
			}
		})
	}
}
