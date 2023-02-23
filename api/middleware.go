package api

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

func panicRecoveryMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("recovered from panic in an HTTP hander: %v", r)
				debug.PrintStack()
				http.Error(w, http.StatusText(500), 500)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	code  int
	wrote bool
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{ResponseWriter: w, code: 200}
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.wrote {
		return
	}
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
	w.wrote = true
	return
}

func (w *loggingResponseWriter) Flush() {
	if fl, ok := w.ResponseWriter.(http.Flusher); ok {
		if w.code == 0 {
			w.WriteHeader(200)
		}
		fl.Flush()
	}
}

func requestLogMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t0 := time.Now()
		lwr := newLoggingResponseWriter(w)
		h.ServeHTTP(lwr, r)
		logger.Printf("%s %s %d %s", r.Method, r.URL.String(), lwr.code, time.Since(t0))
	})
}
