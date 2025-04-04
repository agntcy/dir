// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package webserver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const readHeaderTimeout = 5 * time.Second

func StartLocalServer(h *Handler, port int, errCh chan error) *http.Server {
	r := mux.NewRouter()
	r.HandleFunc("/", h.HandleRequestRedirect).Methods(http.MethodGet).Queries("request", "{request}")
	r.HandleFunc("/", h.HandleCodeRedirect).Methods(http.MethodGet).Queries("code", "{code}")

	server := &http.Server{
		ReadHeaderTimeout: readHeaderTimeout,
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           r,
	}

	go func() {
		errCh <- server.ListenAndServe()
	}()

	return server
}
