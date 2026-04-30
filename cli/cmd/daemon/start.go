// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/agntcy/dir-runtime/discovery"
	discoveryConfig "github.com/agntcy/dir-runtime/discovery/config"
	runtimeserver "github.com/agntcy/dir-runtime/server"
	runtimestore "github.com/agntcy/dir-runtime/store"
	networkinit "github.com/agntcy/dir/cli/cmd/network/init"
	reconciler "github.com/agntcy/dir/reconciler/service"
	"github.com/agntcy/dir/server"
	ocilib "github.com/agntcy/dir/server/store/oci"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/cobra"
	ocistore "oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
)

var logger = logging.Logger("daemon")

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the local directory daemon",
	Long: `Start the gRPC apiserver and reconciler in a single process.

Without --config, built-in defaults (embedded daemon.config.yaml) are used.
When --config is provided, the file is read as the complete configuration.

The daemon blocks until SIGINT or SIGTERM is received.`,
	RunE: runStart,
}

//nolint:cyclop
func runStart(cmd *cobra.Command, _ []string) error {
	running, pid, err := readPID()
	if err != nil {
		return err
	}

	if running {
		return fmt.Errorf("daemon already running (pid %d)", pid)
	}

	if err := os.MkdirAll(opts.DataDir, 0o700); err != nil { //nolint:mnd
		return fmt.Errorf("failed to create data directory %s: %w", opts.DataDir, err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.Server.Routing.KeyPath != "" {
		if err := ensureKeyFile(cfg.Server.Routing.KeyPath); err != nil {
			return fmt.Errorf("failed to ensure peer identity key: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	zotCtx := context.Background()

	if cfg.Server.Store.OCI.LocalDir != "" {
		zotCtx = runEmbeddedZot(ctx, cfg.Server.Store.OCI.RegistryAddress, cfg.Server.Store.OCI.LocalDir)

		cfg.Server.Store.OCI.LocalDir = ""
	}

	// Create API server
	srv, err := server.New(ctx, &cfg.Server)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create ephemeral runtime store
	discoveryErrCh := make(chan error, 1)

	if cfg.Runtime.Enabled {
		logger.Info("Runtime discovery enabled, starting services")

		runtimeStore, err := runtimestore.New(cfg.Runtime.Store)
		if err != nil {
			return fmt.Errorf("failed to create runtime store: %w", err)
		}
		defer runtimeStore.Close()

		// Start discovery service in a separate goroutine
		go func() {
			logger.Info("Starting runtime discovery service for environment", "adapter", cfg.Runtime.Adapter.Type)

			err := discovery.Run(ctx,
				discovery.WithConfig(&discoveryConfig.Config{
					Workers:  discoveryConfig.DefaultWorkers,
					Runtime:  cfg.Runtime.Adapter,
					Resolver: cfg.Runtime.Resolver,
				}),
				discovery.WithStore(runtimeStore),
				discovery.WithLogger(logger.With("runtime", "discovery")),
			)
			if err != nil {
				discoveryErrCh <- fmt.Errorf("runtime discovery service failed: %w", err)
			}
		}()

		// Register runtime API with server
		err = runtimeserver.Register(srv.GRPCServer(), runtimeStore)
		if err != nil {
			return fmt.Errorf("failed to create runtime server: %w", err)
		}
	}

	// Start API server
	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer srv.Close(ctx)

	logger.Info("Server started", "address", cfg.Server.ListenAddress)

	tagLister, err := newTagLister(cfg)
	if err != nil {
		return fmt.Errorf("failed to create OCI tag lister for indexer: %w", err)
	}

	svc, err := reconciler.New(&cfg.Reconciler, srv.Database(), srv.Store(), tagLister)
	if err != nil {
		return fmt.Errorf("failed to create reconciler: %w", err)
	}

	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("failed to start reconciler: %w", err)
	}

	defer func() {
		if err := svc.Stop(); err != nil {
			logger.Error("Failed to stop reconciler", "error", err)
		}
	}()

	logger.Info("Reconciler started")

	if err := writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	defer removePIDFile()

	logger.Info("Daemon ready", "data_dir", opts.DataDir, "config", opts.ConfigFilePath(), "pid", os.Getpid())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Info("Received signal, shutting down", "signal", sig)
	case err := <-discoveryErrCh:
		logger.Error("Runtime discovery service error", "error", err)
	case <-zotCtx.Done():
		logger.Info("zot context cancelled, shutting down")
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down")
	}

	return nil
}

// ensureKeyFile generates a persistent Ed25519 identity key if one does not
// already exist at path. Uses the same PKCS#8 PEM format as `dirctl network init`.
func ensureKeyFile(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat key file: %w", err)
	}

	_, pemData, err := networkinit.GenerateED25519OpenSSLKey()
	if err != nil {
		return fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	if err := os.WriteFile(path, pemData, 0o600); err != nil { //nolint:mnd
		return fmt.Errorf("failed to write key file: %w", err)
	}

	logger.Info("Generated persistent peer identity key", "path", path)

	return nil
}

// newTagLister returns a registry.TagLister for the reconciler's indexer.
// When a local OCI directory is configured, a local oci.Store is opened.
// Otherwise a remote ORAS repository is created from the OCI config.
func newTagLister(cfg *DaemonConfig) (registry.TagLister, error) {
	if dir := cfg.Server.Store.OCI.LocalDir; dir != "" {
		repo, err := ocistore.New(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to open local OCI store: %w", err)
		}

		return repo, nil
	}

	repo, err := ocilib.NewORASRepository(cfg.Server.Store.OCI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote OCI registry: %w", err)
	}

	return repo, nil
}
