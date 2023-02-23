package api

import (
	"errors"
	"log"
	"net/http"
)

func Respond(routes *Routes, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Response", "true")
	}
	if err := routes.Respond(w, r); err != nil {
		log.Printf("responding with route: %v", err)
		if errors.Is(err, ErrNotFound) {
			http.NotFound(w, r)
		} else {
			http.Error(w, http.StatusText(500), 500)
		}
	}
}
