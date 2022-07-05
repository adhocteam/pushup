package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/AdHocRandD/pushup/build"

	"golang.org/x/sync/errgroup"
)

var logger *log.Logger

func main() {
	// FIXME(paulsmith): detect if connected to terminal for VT100 escapes
	logger = log.New(os.Stderr, "[\x1b[36mPUSHUP\x1b[0m] ", 0)

	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	adminPort := flag.String("admin-port", "9090", "port to listen on for admin")

	// FIXME(paulsmith): can't have both port and unixSocket non-empty
	flag.Parse()

	g := new(errgroup.Group)

	g.Go(func() error {
		mux := http.NewServeMux()
		// TODO(paulsmith): allow these middlewares to be configurable on/off
		mux.Handle("/", panicRecoveryMiddleware(requestLogMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := build.Render(w, r); err != nil {
				logger.Printf("rendering route: %v", err)
				if errors.Is(err, build.NotFound) {
					http.NotFound(w, r)
				} else {
					http.Error(w, http.StatusText(500), 500)
				}
				return
			}
		}))))

		var ln net.Listener
		var err error
		if *unixSocket != "" {
			ln, err = net.Listen("unix", *unixSocket)
		} else {
			host := "0.0.0.0"
			addr := host + ":" + *port
			ln, err = net.Listen("tcp4", addr) // TODO(paulsmith): may want to support IPv6
		}
		if err != nil {
			return fmt.Errorf("getting a listener: %v", err)
		}

		srv := http.Server{Handler: mux}

		fmt.Fprintf(os.Stdout, "\x1b[32m↑↑ Pushup ready and listening on %s ↑↑\x1b[0m\n", ln.Addr().String())
		if err := srv.Serve(ln); err != nil {
			return fmt.Errorf("serving HTTP: %v", err)
		}

		return nil
	})

	g.Go(func() error {
		mux := http.NewServeMux()
		mux.Handle("/", http.HandlerFunc(build.Admin))
		srv := http.Server{
			Addr:    "127.0.0.1:" + *adminPort,
			Handler: mux,
		}
		return srv.ListenAndServe()
	})

	if err := g.Wait(); err != nil {
		logger.Fatalf("error: %v", err)
	}
}

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
	return &loggingResponseWriter{w, 200, false}
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.code = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
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
