// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	reconcilerconfig "github.com/agntcy/dir/reconciler/config"
	reconciler "github.com/agntcy/dir/reconciler/service"
	"github.com/agntcy/dir/reconciler/tasks/indexer"
	"github.com/agntcy/dir/reconciler/tasks/name"
	"github.com/agntcy/dir/reconciler/tasks/regsync"
	"github.com/agntcy/dir/reconciler/tasks/signature"
	"github.com/agntcy/dir/server"
	serverconfig "github.com/agntcy/dir/server/config"
	dbconfig "github.com/agntcy/dir/server/database/config"
	namingconfig "github.com/agntcy/dir/server/naming/config"
	publication "github.com/agntcy/dir/server/publication/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	storeconfig "github.com/agntcy/dir/server/store/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/cobra"
	ocistore "oras.land/oras-go/v2/content/oci"
)

var logger = logging.Logger("daemon")

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the local directory daemon",
	Long: `Start the gRPC apiserver and reconciler in a single process with local-mode
defaults (embedded SQLite, filesystem OCI store, all reconciler tasks in-process).

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

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	serverCfg := buildServerConfig()
	reconcilerCfg := buildReconcilerConfig()

	// Start the gRPC server.
	srv, err := server.New(ctx, serverCfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := srv.Start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer srv.Close(ctx)

	logger.Info("Server started", "address", serverCfg.ListenAddress)

	localRepo, err := ocistore.New(opts.StoreDir())
	if err != nil {
		return fmt.Errorf("failed to open local OCI store for indexer: %w", err)
	}

	svc, err := reconciler.New(reconcilerCfg, srv.Database(), srv.Store(), localRepo)
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

	logger.Info("Daemon ready", "data_dir", opts.DataDir, "pid", os.Getpid())

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

func buildServerConfig() *serverconfig.Config {
	return &serverconfig.Config{
		ListenAddress: "localhost:8888",
		Connection:    serverconfig.DefaultConnectionConfig(),
		OASFAPIValidation: serverconfig.OASFAPIValidationConfig{
			SchemaURL: "https://schema.oasf.outshift.com",
		},
		Store: storeconfig.Config{
			Provider: storeconfig.DefaultProvider,
			OCI: ociconfig.Config{
				LocalDir: opts.StoreDir(),
			},
			Verification: storeconfig.VerificationConfig{
				Enabled: storeconfig.DefaultVerificationEnabled,
			},
		},
		Routing: routingconfig.Config{
			ListenAddress:  "/ip4/0.0.0.0/tcp/0",
			BootstrapPeers: []string{},
			DatastoreDir:   opts.RoutingDir(),
			GossipSub: routingconfig.GossipSubConfig{
				Enabled: routingconfig.DefaultGossipSubEnabled,
			},
		},
		Database: dbconfig.Config{
			Type: "sqlite",
			SQLite: dbconfig.SQLiteConfig{
				Path: opts.DBFile(),
			},
		},
		Publication: publication.Config{
			SchedulerInterval: publication.DefaultPublicationSchedulerInterval,
			WorkerCount:       publication.DefaultPublicationWorkerCount,
			WorkerTimeout:     publication.DefaultPublicationWorkerTimeout,
		},
		Naming: namingconfig.Config{
			TTL: namingconfig.DefaultTTL,
		},
	}
}

func buildReconcilerConfig() *reconcilerconfig.Config {
	return &reconcilerconfig.Config{
		Regsync: regsync.Config{
			Enabled: false,
		},
		Indexer: indexer.Config{
			Enabled:  true,
			Interval: indexer.DefaultInterval,
		},
		Signature: signature.Config{
			Enabled:       true,
			Interval:      signature.DefaultInterval,
			TTL:           signature.DefaultTTL,
			RecordTimeout: signature.DefaultRecordTimeout,
		},
		Name: name.Config{
			Enabled:       true,
			Interval:      name.DefaultInterval,
			TTL:           namingconfig.DefaultTTL,
			RecordTimeout: name.DefaultRecordTimeout,
		},
	}
}
