package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOpenTagLexer(t *testing.T) {
	tests := []struct {
		input string
		want  []*attr
	}{
		{
			"<div>",
			[]*attr{},
		},
		{
			"<div disabled>",
			[]*attr{{name: stringPos{"disabled", pos(5)}}},
		},
		{
			`<div class="foo">`,
			[]*attr{{name: stringPos{"class", pos(5)}, value: stringPos{"foo", pos(12)}}},
		},
		{
			`<p   data-^name="/foo/bar/^value"   thing="^asd"  >`,
			[]*attr{
				{
					name: stringPos{
						"data-^name",
						pos(5),
					},
					value: stringPos{
						"/foo/bar/^value",
						pos(17),
					},
				},
				{
					name: stringPos{
						"thing",
						pos(36),
					},
					value: stringPos{
						"^asd",
						pos(43),
					},
				},
			},
		},
	}
	opts := cmp.AllowUnexported(attr{}, stringPos{})
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got, err := scanAttrs(tt.input)
			if err != nil {
				t.Fatalf("scanAttrs: %v", err)
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
