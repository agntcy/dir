// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the reconciler service.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/service"
	"github.com/agntcy/dir/reconciler/tasks/regsync"
	"github.com/agntcy/dir/server/database"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/utils/logging"
)

const (
	// defaultHealthPort is the default port for the health check endpoint.
	defaultHealthPort = ":8080"

	// healthCheckTimeout is the timeout for health check operations.
	healthCheckTimeout = 5 * time.Second
)

var logger = logging.Logger("reconciler")

func main() {
	if err := run(); err != nil {
		logger.Error("Reconciler failed", "error", err)
		os.Exit(1)
	}
}

//nolint:wrapcheck
func run() error {
	logger.Info("Starting reconciler service")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	// Create database connection
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create service
	svc := service.New()

	// Register tasks
	if cfg.Regsync.Enabled {
		regsyncTask, err := regsync.NewTask(cfg.Regsync, cfg.LocalRegistry, db)
		if err != nil {
			return err
		}

		svc.RegisterTask(regsyncTask)
	}

	// Create context that listens for signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health check server with database readiness check
	healthServer := startHealthServer(func(ctx context.Context) bool {
		return db.IsReady(ctx)
	})

	// Start the service
	if err := svc.Start(ctx); err != nil {
		return err
	}

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sigCh
	logger.Info("Received signal, shutting down", "signal", sig)

	// Cancel context to stop tasks
	cancel()

	// Stop the service
	if err := svc.Stop(); err != nil {
		logger.Error("Failed to stop service gracefully", "error", err)
	}

	// Shutdown health server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), healthCheckTimeout)
	defer shutdownCancel()

	if err := healthServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown health server", "error", err)
	}

	logger.Info("Reconciler service stopped")

	return nil
}

// startHealthServer starts a simple HTTP health check server.
func startHealthServer(readinessCheck func(ctx context.Context) bool) *http.Server {
	mux := http.NewServeMux()

	// Liveness probe - always returns OK if the process is running
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Readiness probe - checks database connectivity
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		if readinessCheck(ctx) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	})

	port := os.Getenv("HEALTH_PORT")
	if port == "" {
		port = defaultHealthPort
	}

	server := &http.Server{Addr: port, Handler: mux, ReadHeaderTimeout: healthCheckTimeout}

	go func() {
		logger.Info("Starting health check server", "address", port)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("Health check server error", "error", err)
		}
	}()

	return server
}
