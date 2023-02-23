package api

import (
	"fmt"
	"html/template"
	"io"
	"strconv"
)

func PrintEscaped(w io.Writer, val any) {
	switch val := val.(type) {
	case string:
		//nolint:errcheck
		io.WriteString(w, template.HTMLEscapeString(val))
	case fmt.Stringer:
		//nolint:errcheck
		io.WriteString(w, template.HTMLEscapeString(val.String()))
	case []byte:
		template.HTMLEscape(w, val)
	case int:
		//nolint:errcheck
		io.WriteString(w, strconv.Itoa(val))
	case template.HTML:
		//nolint:errcheck
		io.WriteString(w, string(val))
	default:
		//nolint:errcheck
		io.WriteString(w, template.HTMLEscapeString(fmt.Sprint(val)))
	}
}
