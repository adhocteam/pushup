package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOpenTagLexer(t *testing.T) {
	tests := []struct {
		input string
		want  []*Attr
	}{
		{
			"<div>",
			[]*Attr{},
		},
		{
			"<div disabled>",
			[]*Attr{{Name: StringPos{"disabled", pos(5)}}},
		},
		{
			`<div class="foo">`,
			[]*Attr{{Name: StringPos{"class", pos(5)}, Value: StringPos{"foo", pos(12)}}},
		},
		{
			`<p   data-^name="/foo/bar/^value"   thing="^asd"  >`,
			[]*Attr{
				{
					Name: StringPos{
						"data-^name",
						pos(5),
					},
					Value: StringPos{
						"/foo/bar/^value",
						pos(17),
					},
				},
				{
					Name: StringPos{
						"thing",
						pos(36),
					},
					Value: StringPos{
						"^asd",
						pos(43),
					},
				},
			},
		},
	}
	opts := cmp.AllowUnexported(Attr{}, StringPos{})
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
