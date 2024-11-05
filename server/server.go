package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
)

type Server struct {
	*http.Server
}

func New(addr string, handler http.Handler) *Server {
	return &Server{
		Server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

func (s *Server) ListenAndServe() error {
	if fdStr := os.Getenv("PUSHUP_LISTENER_FD"); fdStr != "" {
		fd, err := strconv.Atoi(fdStr)
		if err != nil {
			return fmt.Errorf("converting listener FD string to int: %w", err)
		}

		file := os.NewFile(uintptr(fd), "pushup-proxy-sock")
		ln, err := net.FileListener(file)
		if err != nil {
			return fmt.Errorf("getting a listener: %w", err)
		}
		file.Close()
		return s.Serve(ln)
	}

	return s.Server.ListenAndServe()
}
