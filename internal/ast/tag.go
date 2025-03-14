package ast

import (
	"bytes"
	"html"
)

// NOTE: Consider separate StartTag and EndTag types since attributes are only
// applicable to start tags.

// A Tag represents an HTML start or end tag name and attributes.
type Tag struct {
	Name  string
	Attrs []*Attr
}

func (t Tag) String() string {
	if len(t.Attrs) == 0 {
		return t.Name
	}
	buf := bytes.NewBufferString(t.Name)
	for _, a := range t.Attrs {
		buf.WriteByte(' ')
		buf.WriteString(a.Name.Text)
		buf.WriteString(`="`)
		buf.WriteString(html.EscapeString(a.Value.Text))
		buf.WriteByte('"')
	}
	return buf.String()
}

func (t Tag) Start() string {
	return "<" + t.String() + ">"
}

func (t Tag) End() string {
	return "</" + t.Name + ">"
}

func NewTag(tagname []byte, attrs []*Attr) Tag {
	return Tag{Name: string(tagname), Attrs: attrs}
}
