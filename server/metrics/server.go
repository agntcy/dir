// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package metrics provides Prometheus metrics collection infrastructure.
// It manages a separate HTTP server for exposing metrics on the /metrics endpoint.
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/agntcy/dir/utils/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var logger = logging.Logger("metrics")

const (
	// metricsCollectionTimeout is the timeout for collecting metrics.
	// Prevents the /metrics endpoint from hanging indefinitely.
	metricsCollectionTimeout = 10 * time.Second

	// httpReadTimeout is the maximum duration for reading the entire request.
	httpReadTimeout = 10 * time.Second

	// httpReadHeaderTimeout is the maximum duration for reading request headers.
	httpReadHeaderTimeout = 5 * time.Second

	// httpWriteTimeout is the maximum duration for writing the response.
	httpWriteTimeout = 30 * time.Second

	// httpIdleTimeout is the maximum duration to wait for the next request.
	httpIdleTimeout = 60 * time.Second

	// serverStartupDelay is the delay after starting the server to check for errors.
	serverStartupDelay = 100 * time.Millisecond
)

// Server manages the Prometheus metrics HTTP server.
// It provides a separate HTTP endpoint for metrics collection,
// independent of the main gRPC server.
type Server struct {
	registry *prometheus.Registry
	server   *http.Server
	address  string
}

// New creates a new metrics server with a custom Prometheus registry.
// The server listens on the specified address (e.g., ":9090").
// Call Start() to begin serving metrics.
func New(address string) *Server {
	// Create custom registry to avoid conflicts with global registry
	registry := prometheus.NewRegistry()

	// Create HTTP handler for Prometheus metrics
	// Using promhttp.HandlerFor allows us to use our custom registry
	metricsHandler := promhttp.HandlerFor(
		registry,
		promhttp.HandlerOpts{
			// Enable OpenMetrics format for better compatibility
			EnableOpenMetrics: true,
			// Timeout for collecting metrics (prevent hanging)
			Timeout: metricsCollectionTimeout,
		},
	)

	// Create HTTP mux for routing
	mux := http.NewServeMux()
	mux.Handle("/metrics", metricsHandler)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    address,
		Handler: mux,
		// Reasonable timeouts to prevent resource exhaustion
		ReadTimeout:       httpReadTimeout,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
	}

	return &Server{
		registry: registry,
		server:   httpServer,
		address:  address,
	}
}

// Registry returns the Prometheus registry for registering custom metrics.
func (s *Server) Registry() *prometheus.Registry {
	return s.registry
}

// Start starts the HTTP server in the background.
// Returns immediately after starting the server goroutine.
func (s *Server) Start() error {
	// Start HTTP server in background goroutine
	go func() {
		logger.Info("Metrics server starting", "address", s.address)

		// ListenAndServe blocks until server is shut down
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// ErrServerClosed is expected during graceful shutdown
			logger.Error("Metrics server error", "error", err)
		}
	}()

	// Give the server a moment to start and check for immediate errors
	// (e.g., port already in use)
	time.Sleep(serverStartupDelay)

	logger.Info("Metrics server started successfully", "address", s.address, "endpoint", "/metrics")

	return nil
}

// Stop gracefully shuts down the HTTP server.
// Waits for in-flight requests to complete up to the context timeout.
func (s *Server) Stop(ctx context.Context) error {
	logger.Info("Stopping metrics server", "address", s.address)

	// Gracefully shutdown the server, waiting for in-flight requests to complete
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown metrics server: %w", err)
	}

	logger.Info("Metrics server stopped successfully")

	return nil
}
