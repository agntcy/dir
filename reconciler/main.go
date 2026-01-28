// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package main is the entry point for the reconciler service.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/service"
	"github.com/agntcy/dir/reconciler/tasks/indexer"
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

	// Initialize OASF validator for record validation
	if cfg.SchemaURL != "" {
		if err := corev1.InitializeValidator(cfg.SchemaURL); err != nil {
			return fmt.Errorf("failed to initialize OASF validator: %w", err)
		}

		logger.Info("OASF validator initialized", "schema_url", cfg.SchemaURL)
	} else {
		logger.Warn("OASF schema URL not configured, record validation will be skipped")
	}

	// Create database connection
	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create OCI store for accessing the local registry
	store, err := oci.New(cfg.LocalRegistry)
	if err != nil {
		return err
	}

	// Create ORAS repository client for registry operations (e.g., listing tags)
	repo, err := oci.NewORASRepository(cfg.LocalRegistry)
	if err != nil {
		return err
	}

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

	if cfg.Indexer.Enabled {
		indexerTask, err := indexer.NewTask(cfg.Indexer, cfg.LocalRegistry, store, repo, db)
		if err != nil {
			return err
		}

		svc.RegisterTask(indexerTask)
	}

	// Create context that listens for signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start health check server with database and store readiness check
	healthServer := startHealthServer(func(ctx context.Context) bool {
		return db.IsReady(ctx) && store.IsReady(ctx)
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
