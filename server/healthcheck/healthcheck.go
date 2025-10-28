// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package healthcheck provides simple HTTP health check endpoints.
package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("healthcheck")

// CheckFunc is a function that performs a health check.
// Return true if healthy, false otherwise.
type CheckFunc func(ctx context.Context) bool

// Checker manages health checks and HTTP endpoints.
type Checker struct {
	mu              sync.RWMutex
	readinessChecks map[string]CheckFunc
	server          *http.Server
}

// Response is the JSON response format.
type Response struct {
	Status  string            `json:"status"`
	Checks  map[string]string `json:"checks,omitempty"`
	Message string            `json:"message,omitempty"`
}

// New creates a new health checker.
func New() *Checker {
	return &Checker{
		readinessChecks: make(map[string]CheckFunc),
	}
}

// AddReadinessCheck adds a readiness check.
func (c *Checker) AddReadinessCheck(name string, check CheckFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.readinessChecks[name] = check
}

// Start starts the health check HTTP server.
func (c *Checker) Start(address string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz/live", c.handleLiveness)
	mux.HandleFunc("/healthz/ready", c.handleReadiness)
	mux.HandleFunc("/livez", c.handleLiveness)
	mux.HandleFunc("/readyz", c.handleReadiness)

	//nolint:mnd
	c.server = &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	go func() {
		logger.Info("Starting health check server", "address", address)

		if err := c.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Health check server error", "error", err)
		}
	}()

	return nil
}

// Stop gracefully stops the health check server.
func (c *Checker) Stop(ctx context.Context) error {
	if c.server == nil {
		return nil
	}

	logger.Info("Stopping health check server")

	return c.server.Shutdown(ctx) //nolint:wrapcheck
}

// handleLiveness handles liveness probe requests.
// Liveness checks if the application is running (always returns 200 OK).
func (c *Checker) handleLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(Response{
		Status:  "ok",
		Message: "server is alive",
	}); err != nil {
		logger.Error("Failed to encode liveness response", "error", err)
	}
}

// handleReadiness handles readiness probe requests.
// Readiness checks if the application is ready to serve traffic.
func (c *Checker) handleReadiness(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()

	checks := make(map[string]CheckFunc, len(c.readinessChecks))
	for name, check := range c.readinessChecks {
		checks[name] = check
	}

	c.mu.RUnlock()

	// Run all checks with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second) //nolint:mnd
	defer cancel()

	results := make(map[string]string)
	allHealthy := true

	for name, check := range checks {
		if check(ctx) {
			results[name] = "pass" //nolint:goconst
		} else {
			results[name] = "fail"
			allHealthy = false
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if allHealthy {
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(Response{
			Status: "ready",
			Checks: results,
		}); err != nil {
			logger.Error("Failed to encode readiness response", "error", err)
		}
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)

		if err := json.NewEncoder(w).Encode(Response{
			Status: "not ready",
			Checks: results,
		}); err != nil {
			logger.Error("Failed to encode readiness response", "error", err)
		}
	}
}
