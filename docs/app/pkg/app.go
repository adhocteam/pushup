package build

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

func convertMarkdown(text []byte) template.HTML {
	var buf bytes.Buffer
	if err := goldmark.Convert(text, &buf); err != nil {
		panic(err)
	}
	return template.HTML(buf.String())
}
