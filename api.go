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

type PushupContext interface {
	Request() *http.Request
	Writer() http.ResponseWriter
	Params() map[string]any
	Children() func(PushupContext)
}

type ResponseWriter struct {
	http.ResponseWriter
	buf     *bytes.Buffer
	discard bool
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	var discard bool
	if rw, ok := w.(*ResponseWriter); ok {
		discard = rw.discard
	}
	return &ResponseWriter{
		ResponseWriter: w,
		buf:            new(bytes.Buffer),
		discard:        discard,
	}
}

func (w *ResponseWriter) Write(b []byte) (int, error) {
	if w.discard {
		return len(b), nil
	}
	return w.buf.Write(b)
}

func (w *ResponseWriter) flush() (int64, error) {
	if rw, ok := w.ResponseWriter.(*ResponseWriter); ok {
		return w.buf.WriteTo(rw.buf)
	}
	return w.buf.WriteTo(w.ResponseWriter)
}

func (w *ResponseWriter) Flush() {
	_, err := w.flush()
	if err != nil {
		panic(fmt.Sprintf("unexpected error flushing buffer to underlying response writer: %v", err))
	}
}

func (w *ResponseWriter) FlushError() error {
	_, err := w.flush()
	return err
}

func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *ResponseWriter) SetDiscard() {
	w.discard = true
}

func (w *ResponseWriter) UnsetDiscard() {
	w.discard = false
}

func Param(name string, req *http.Request, props map[string]any) any {
	if val, ok := props[name]; ok {
		return val
	}
	if val := req.PathValue(name); val != "" {
		return val
	}
	return req.FormValue(name)
}
