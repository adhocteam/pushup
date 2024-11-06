package pushup

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/http"
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

type UserContext interface {
	Request() *http.Request
	Writer() http.ResponseWriter
	Params() map[string]any
	Children() func(UserContext)
}

type pushupResponseWriter struct {
	http.ResponseWriter
	buf *bytes.Buffer
}

func NewResponseWriter(w http.ResponseWriter) *pushupResponseWriter {
	return &pushupResponseWriter{
		ResponseWriter: w,
		buf:            new(bytes.Buffer),
	}
}

func (w *pushupResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}

func (w *pushupResponseWriter) Flush() {
	_, err := w.flush()
	if err != nil {
		panic(fmt.Sprintf("unexpected error flushing buffer to underlying response writer: %v", err))
	}
}

func (w *pushupResponseWriter) FlushError() error {
	_, err := w.flush()
	return err
}

func (w *pushupResponseWriter) flush() (int64, error) {
	return w.buf.WriteTo(w.ResponseWriter)
}

func (w *pushupResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func Param(name string, req *http.Request, props map[string]any) any {
	if val, ok := props[name]; ok {
		return val
	}
	return req.FormValue(name)
}
