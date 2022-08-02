package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/AdHocRandD/pushup/build"
)

var logger *log.Logger

func main() {
	// FIXME(paulsmith): detect if connected to terminal for VT100 escapes
	logger = log.New(os.Stderr, "[\x1b[36mPUSHUP\x1b[0m] ", 0)

	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	//adminPort := flag.String("admin-port", "9090", "port to listen on for admin")

	// FIXME(paulsmith): can't have both port and unixSocket non-empty
	flag.Parse()

	mux := http.NewServeMux()
	// TODO(paulsmith): allow these middlewares to be configurable on/off
	var h http.Handler = http.HandlerFunc(pushupHandler)
	h = requestLogMiddleware(h)
	h = panicRecoveryMiddleware(h)
	mux.Handle("/", h)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s\n", r.RequestURI)
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=7776000")
		fmt.Fprintln(w, "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQEAYAAABPYyMiAAAABmJLR0T///////8JWPfcAAAACXBIWXMAAABIAAAASABGyWs+AAAAF0lEQVRIx2NgGAWjYBSMglEwCkbBSAcACBAAAeaR9cIAAAAASUVORK5CYII=")
	})

	var ln net.Listener
	var err error
	if parentFd := os.Getenv("PUSHUP_LISTENER_FD"); parentFd != "" {
		fd, err := strconv.Atoi(parentFd)
		if err != nil {
			log.Fatalf("converting %q to int: %v", parentFd, err)
		}
		ln, err = net.FileListener(os.NewFile(uintptr(fd), "pushup-parent-sock"))
	} else {
		if *unixSocket != "" {
			ln, err = net.Listen("unix", *unixSocket)
		} else {
			host := "0.0.0.0"
			addr := host + ":" + *port
			ln, err = net.Listen("tcp4", addr) // TODO(paulsmith): may want to support IPv6
		}
	}
	if err != nil {
		logger.Fatalf("getting a listener: %v", err)
	}
	defer ln.Close()

	srv := http.Server{
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	// NOTE(paulsmith): keep this in sync with the string in main_test.go in the compiler
	fmt.Fprintf(os.Stdout, "\x1b[32m↑↑ Pushup ready and listening on %s ↑↑\x1b[0m\n", ln.Addr().String())

	go srv.Serve(ln)

	//<-time.After(1 * time.Second)
	//logger.Printf("SOME ERROR!")
	//os.Exit(0x55)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()

	stop()
	logger.Printf("shutting down gracefully, press Ctrl+C to force immediate")

	{
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			logger.Printf("server shutdown: %v", err)
		}
		logger.Printf("shutdown complete")
	}
}

func pushupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Response", "true")
	}
	if err := build.Render(w, r); err != nil {
		logger.Printf("rendering route: %v", err)
		if errors.Is(err, build.NotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
		return
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
