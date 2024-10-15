package element

import (
	"testing"

	"github.com/adhocteam/pushup/internal/source"
)

func TestTagString(t *testing.T) {
	tests := []struct {
		name string
		tag  Tag
		want string
	}{
		{
			name: "Tag with no attributes",
			tag:  NewTag([]byte("div"), nil),
			want: "div",
		},
		{
			name: "Tag with one attribute",
			tag: NewTag([]byte("a"), []*Attr{
				{Name: source.StringPos{Text: "href"}, Value: source.StringPos{Text: "https://example.com"}},
			}),
			want: `a href="https://example.com"`,
		},
		{
			name: "Tag with multiple attributes",
			tag: NewTag([]byte("img"), []*Attr{
				{Name: source.StringPos{Text: "src"}, Value: source.StringPos{Text: "image.png"}},
				{Name: source.StringPos{Text: "alt"}, Value: source.StringPos{Text: "An image"}},
			}),
			want: `img src="image.png" alt="An image"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tag.String()
			if got != tt.want {
				t.Errorf("Tag.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTagStart(t *testing.T) {
	tag := NewTag([]byte("div"), nil)
	want := "<div>"

	if got := tag.Start(); got != want {
		t.Errorf("Tag.Start() = %v, want %v", got, want)
	}
}

func TestTagEnd(t *testing.T) {
	tag := NewTag([]byte("div"), nil)
	want := "</div>"

	if got := tag.End(); got != want {
		t.Errorf("Tag.End() = %v, want %v", got, want)
	}
}
