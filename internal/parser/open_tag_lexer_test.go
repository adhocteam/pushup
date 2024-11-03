package parser

import (
	"testing"

	"github.com/adhocteam/pushup/internal/ast"
	"github.com/adhocteam/pushup/internal/source"
	"github.com/google/go-cmp/cmp"
)

func TestOpenTagLexer(t *testing.T) {
	tests := []struct {
		input string
		want  []*ast.Attr
	}{
		{
			"<div>",
			[]*ast.Attr{},
		},
		{
			"<div disabled>",
			[]*ast.Attr{{Name: source.StringPos{"disabled", source.Pos(5)}}},
		},
		{
			`<div class="foo">`,
			[]*ast.Attr{{Name: source.StringPos{"class", source.Pos(5)}, Value: source.StringPos{"foo", source.Pos(12)}}},
		},
		{
			`<p   data-^name="/foo/bar/^value"   thing="^asd"  >`,
			[]*ast.Attr{
				{
					Name: source.StringPos{
						"data-^name",
						source.Pos(5),
					},
					Value: source.StringPos{
						"/foo/bar/^value",
						source.Pos(17),
					},
				},
				{
					Name: source.StringPos{
						"thing",
						source.Pos(36),
					},
					Value: source.StringPos{
						"^asd",
						source.Pos(43),
					},
				},
			},
		},
	}
	opts := cmp.AllowUnexported(ast.Attr{}, source.StringPos{})
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := scanAttrs(tt.input)
			if err != nil {
				t.Fatalf("scanast.Attrs: %v", err)
			}
			if diff := cmp.Diff(tt.want, got, opts); diff != "" {
				t.Errorf("(-want, +got)\n%s", diff)
			}
		})
	}
}

func FuzzOpenTagLexer(f *testing.F) {
	seeds := []string{
		"<a href=\"https://adhoc.team/\">",
		"<b>",
		"<input checked>",
	}

	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, in []byte) {
		_, err := scanAttrs(string(in))
		if err != nil {
			if _, ok := err.(openTagScanError); !ok {
				t.Errorf("expected scan error, got %T %v", err, err)
			}
		}
	})
}
