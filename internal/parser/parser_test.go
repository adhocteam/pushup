package parser

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/element"
	"github.com/adhocteam/pushup/internal/source"
	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update golden files")

func TestParser(t *testing.T) {
	testCases, err := filepath.Glob("testdata/*.up")
	if err != nil {
		t.Fatal(err)
	}
	for _, inputFile := range testCases {
		t.Run(filepath.Base(inputFile), func(t *testing.T) {
			input, err := os.ReadFile(inputFile)
			if err != nil {
				t.Fatalf("failed to read input file: %v", err)
			}

			actual, err := parse(string(input))
			if err != nil {
				t.Fatalf("unexpected error parsing input: %v", err)
			}

			goldenFile := inputFile[:len(inputFile)-len(".up")] + ".json"

			if *update {
				actualJSON, err := json.MarshalIndent(actual, "", "    ")
				if err != nil {
					t.Fatalf("failed to marshal actual result: %v", err)
				}

				if err := os.WriteFile(goldenFile, actualJSON, 0644); err != nil {
					t.Fatalf("failed to update golden file: %v", err)
				}
			} else {
				expectedJSON, err := os.ReadFile(goldenFile)
				if err != nil {
					t.Fatalf("failed to read golden file: %v", err)
				}

				var expected ast.Document
				if err := json.Unmarshal(expectedJSON, &expected); err != nil {
					t.Fatalf("failed to unmarshal golden file: %v", err)
				}

				if diff := cmp.Diff(&expected, actual); diff != "" {
					t.Errorf("unexpected parse result (-expected +actual):\n%s", diff)
				}
			}
		})
	}
}

func TestParseSyntaxErrors(t *testing.T) {
	tests := []struct {
		input string
		// expected error conditions
		lineNo int
		column int
	}{
		{"^if", 1, 4},
		{
			`^if true {
	<illegal />
}`, 2, 13,
		},
		// FIXME(paulsmith): add more syntax errors
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			tree, err := parse(tt.input)
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
		_, err := parse(string(in))
		if err != nil {
			if _, ok := err.(syntaxError); !ok {
				t.Errorf("expected syntax error, got %T %v", err, err)
			}
		}
	})
}

func TestTagString(t *testing.T) {
	tests := []struct {
		tag  element.Tag
		want string
	}{
		{
			element.Tag{Name: "h1"},
			"h1",
		},
		{
			element.Tag{Name: "div", Attrs: []*element.Attr{{Name: source.StringPos{Text: "class"}, Value: source.StringPos{Text: "banner"}}}},
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
