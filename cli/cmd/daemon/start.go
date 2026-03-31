// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	reconciler "github.com/agntcy/dir/reconciler/service"
	"github.com/agntcy/dir/server"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/cobra"
	ocistore "oras.land/oras-go/v2/content/oci"
)

var logger = logging.Logger("daemon")

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the local directory daemon",
	Long: `Start the gRPC apiserver and reconciler in a single process.

Configuration is read from a YAML file (default: <data-dir>/daemon.config.yaml).
If the file does not exist, a default config is written automatically.
Use --config to specify a custom path.

The daemon blocks until SIGINT or SIGTERM is received.`,
	RunE: runStart,
}

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

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	srv, err := server.New(ctx, &cfg.Server)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer srv.Close(ctx)

	logger.Info("Server started", "address", cfg.Server.ListenAddress)

	localRepo, err := ocistore.New(cfg.Server.Store.OCI.LocalDir)
	if err != nil {
		return fmt.Errorf("failed to open local OCI store for indexer: %w", err)
	}

	svc, err := reconciler.New(&cfg.Reconciler, srv.Database(), srv.Store(), localRepo)
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
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down")
	}

	return nil
}
