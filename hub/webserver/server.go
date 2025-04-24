// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package webserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/gorilla/mux"
)

const (
	readHeaderTimeout  = 5 * time.Second
	numRetries         = 5
	timeBetweenRetries = 1 * time.Second
)

func StartLocalServer(h *Handler, port int, errCh chan error) (*http.Server, error) {
	r := mux.NewRouter()
	r.HandleFunc("/healthz", h.HandleHealthz).Methods(http.MethodGet)
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

	var err error

	for range numRetries {
		time.Sleep(timeBetweenRetries)

		var req *http.Request

		req, err = http.NewRequestWithContext(context.Background(), http.MethodGet, fmt.Sprintf("http://localhost:%d/healthz", port), nil)
		if err != nil {
			continue
		}

		var resp *http.Response

		resp, err = httpUtils.CreateSecureHTTPClient().Do(req)
		if err != nil {
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("server returned unexpected status code: %d", resp.StatusCode)

			continue
		}

		return server, err
	}

	return nil, fmt.Errorf("failed to start server after %d retries: %w", numRetries, err)
}
