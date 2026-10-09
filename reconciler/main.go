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

	"github.com/agntcy/dir/client"
	"github.com/agntcy/dir/reconciler/config"
	"github.com/agntcy/dir/reconciler/recordevents"
	"github.com/agntcy/dir/reconciler/routing"
	"github.com/agntcy/dir/reconciler/service"
	"github.com/agntcy/dir/server/database"
	"github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/server/validators"
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
	// "dry-run" tries a candidate policy against the node's records and exits;
	// anything else starts the reconciler.
	if len(os.Args) > 1 && os.Args[1] == dryRunCommand {
		if err := runDryRun(os.Args[2:]); err != nil {
			logger.Error("Dry run failed", "error", err)
			os.Exit(1)
		}

		return
	}

	if err := run(); err != nil {
		logger.Error("Reconciler failed", "error", err)
		os.Exit(1)
	}
}

//nolint:wrapcheck,cyclop
func run() error {
	logger.Info("Starting reconciler service")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	validatorRegistry, err := validators.NewRegistry(context.Background(), cfg.Policy.Validators, cfg.Policy.Dir)
	if err != nil {
		return err
	}

	// Create database connection
	db, err := database.New(cfg.Database)
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

	// Create the gRPC routing client. In standalone mode the routing layer
	// (Badger datastore) lives inside the server process and cannot be shared
	// across process boundaries, so we connect to the server over gRPC instead.
	// If no server address is configured the metrics and sync tasks
	// are skipped.
	var routingAPI service.RoutingAPI

	// The same connection tells the reconciler when a record is pushed, so the
	// indexer need not wait for its interval.
	var records recordevents.Source

	if cfg.ServerAddress != "" { //nolint:nestif
		// Build a client.Config from the server address and authn settings.
		serverClientCfg := &client.Config{
			ServerAddress: cfg.ServerAddress,
			AuthMode:      "insecure",
		}

		if cfg.ServerAuthn.Enabled {
			serverClientCfg.AuthMode = string(cfg.ServerAuthn.Mode)
			serverClientCfg.SpiffeSocketPath = cfg.ServerAuthn.SocketPath

			if len(cfg.ServerAuthn.Audiences) > 0 {
				serverClientCfg.JWTAudience = cfg.ServerAuthn.Audiences[0]
			}
		}

		dirClient, err := client.New(context.Background(), client.WithConfig(serverClientCfg))
		if err != nil {
			return fmt.Errorf("failed to connect to apiserver for provider counts: %w", err)
		}

		defer dirClient.Close()

		routingAPI = routing.NewFromClient(dirClient.RoutingServiceClient)
		records = recordevents.NewClientSource(dirClient)

		logger.Info("Routing connected to apiserver", "address", cfg.ServerAddress)
	} else {
		logger.Warn("server_address not configured; metrics and sync tasks will be skipped")
	}

	svc, err := service.New(cfg, db, store, repo, validatorRegistry, routingAPI)
	if err != nil {
		return err
	}

	svc.WatchRecords(records)

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
