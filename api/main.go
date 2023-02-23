package api

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var logger *log.Logger

func Main(pushupHandler http.Handler) {
	// FIXME(paulsmith): detect if connected to terminal for VT100 escapes
	logger = log.New(os.Stderr, "[\x1b[36mPUSHUP\x1b[0m] ", 0)

	port := flag.String("port", "8080", "port to listen on with TCP IPv4")
	unixSocket := flag.String("unix-socket", "", "path to listen on with Unix socket")
	//adminPort := flag.String("admin-port", "9090", "port to listen on for admin")

	// FIXME(paulsmith): can't have both port and unixSocket non-empty
	flag.Parse()

	mux := http.NewServeMux()
	// TODO(paulsmith): allow these middlewares to be configurable on/off
	var h http.Handler = pushupHandler
	h = requestLogMiddleware(h)
	h = http.TimeoutHandler(h, 5*time.Second, "")
	h = panicRecoveryMiddleware(h)
	mux.Handle("/", h)
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s\n", r.RequestURI)
		w.Header().Set("Content-Type", "image/x-icon")
		w.Header().Set("Cache-Control", "public, max-age=7776000")
		fmt.Fprintln(w, "data:image/x-icon;base64,iVBORw0KGgoAAAANSUhEUgAAABAAAAAQEAYAAABPYyMiAAAABmJLR0T///////8JWPfcAAAACXBIWXMAAABIAAAASABGyWs+AAAAF0lEQVRIx2NgGAWjYBSMglEwCkbBSAcACBAAAeaR9cIAAAAASUVORK5CYII=")
	})
	//TODO
	// build.AddStaticHandler(mux)
	// pprof
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

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
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	// NOTE(paulsmith): keep this in sync with the string in main_test.go in the compiler
	fmt.Fprintf(os.Stdout, "\x1b[32m↑↑ Pushup ready and listening on %s ↑↑\x1b[0m\n", ln.Addr().String())

	go srv.Serve(ln)

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
