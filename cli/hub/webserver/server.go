package webserver

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func StartLocalServer(h *Handler, port int, errCh chan error) *http.Server {
	r := mux.NewRouter()
	r.HandleFunc("/", h.HandleRequestRedirect).Methods(http.MethodGet).Queries("request", "{request}")
	r.HandleFunc("/", h.HandleCodeRedirect).Methods(http.MethodGet).Queries("code", "{code}")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	go func() {
		errCh <- server.ListenAndServe()
	}()

	return server
}
