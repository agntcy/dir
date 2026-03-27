// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/agntcy/dir/cli/presenter"
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
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the local directory daemon",
	Long: `Start the gRPC apiserver and reconciler in a single process with local-mode
defaults (embedded SQLite, filesystem OCI store, all reconciler tasks in-process).

The daemon blocks until SIGINT or SIGTERM is received.`,
	RunE: runStart,
}

func runStart(cmd *cobra.Command, _ []string) error {
	if err := checkNotRunning(); err != nil {
		return err
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil { //nolint:mnd
		return fmt.Errorf("failed to create data directory %s: %w", dataDir, err)
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

	presenter.Println(cmd, "Server started", "address", serverCfg.ListenAddress)

	// Start the reconciler using the server's DB and store.
	svc, err := reconciler.New(reconcilerCfg, srv.Database(), srv.Store(), nil)
	if err != nil {
		return fmt.Errorf("failed to create reconciler: %w", err)
	}

	if err := svc.Start(ctx); err != nil {
		return fmt.Errorf("failed to start reconciler: %w", err)
	}

	defer func() {
		if err := svc.Stop(); err != nil {
			presenter.Error(cmd, "Failed to stop reconciler", "error", err)
		}
	}()

	presenter.Println(cmd, "Reconciler started")

	// Write PID file.
	if err := writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	defer removePIDFile(cmd)

	presenter.Println(cmd, "Daemon ready", "data_dir", dataDir, "pid", os.Getpid())

	// Block until signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		presenter.Println(cmd, "Received signal, shutting down", "signal", sig)
	case <-ctx.Done():
		presenter.Println(cmd, "Context cancelled, shutting down")
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
				LocalDir: filepath.Join(dataDir, "store"),
			},
			Verification: storeconfig.VerificationConfig{
				Enabled: storeconfig.DefaultVerificationEnabled,
			},
		},
		Routing: routingconfig.Config{
			ListenAddress:  "/ip4/0.0.0.0/tcp/0",
			BootstrapPeers: []string{},
			DatastoreDir:   filepath.Join(dataDir, "routing"),
			GossipSub: routingconfig.GossipSubConfig{
				Enabled: routingconfig.DefaultGossipSubEnabled,
			},
		},
		Database: dbconfig.Config{
			Type: "sqlite",
			SQLite: dbconfig.SQLiteConfig{
				Path: filepath.Join(dataDir, "dir.db"),
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
			Enabled:    true,
			Interval:   regsync.DefaultInterval,
			BinaryPath: regsync.DefaultBinaryPath,
			Timeout:    regsync.DefaultTimeout,
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

func checkNotRunning() error {
	pid, err := readPID()
	if err != nil {
		return nil //nolint:nilerr // no PID file means no daemon running
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil //nolint:nilerr // process lookup failure means no daemon running
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return nil //nolint:nilerr // signal failure means process is not alive
	}

	return fmt.Errorf("daemon already running (pid %d)", pid)
}

func writePIDFile() error {
	if err := os.WriteFile(pidFilePath(), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil { //nolint:mnd
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

func removePIDFile(cmd *cobra.Command) {
	if err := os.Remove(pidFilePath()); err != nil && !os.IsNotExist(err) {
		presenter.Error(cmd, "Failed to remove PID file", "error", err)
	}
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file: %w", err)
	}

	return pid, nil
}
