// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	zotapi "zotregistry.dev/zot/v2/pkg/api"
	zotconfig "zotregistry.dev/zot/v2/pkg/api/config"
)

// Options holds all daemon path configuration.
type Options struct {
	DataDir    string
	ConfigFile string
}

func (o *Options) DBFile() string     { return filepath.Join(o.DataDir, "dir.db") }
func (o *Options) RoutingDir() string { return filepath.Join(o.DataDir, "routing") }
func (o *Options) PIDFile() string    { return filepath.Join(o.DataDir, "daemon.pid") }

// ConfigFilePath returns the config file path.
func (o *Options) ConfigFilePath() string {
	if o.ConfigFile != "" {
		return o.ConfigFile
	}

	return filepath.Join(o.DataDir, DefaultConfigFile)
}

func defaultDataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), ".agntcy", "dir")
	}

	return filepath.Join(home, ".agntcy", "dir")
}

var opts = &Options{}

// readPID reads the PID file and probes the process.
func readPID() (bool, int, error) {
	data, readErr := os.ReadFile(opts.PIDFile())
	if readErr != nil {
		return false, 0, nil //nolint:nilerr // no PID file means no daemon
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return false, 0, fmt.Errorf("invalid PID file: %w", err)
	}

	proc, findErr := os.FindProcess(pid)
	if findErr != nil {
		return false, pid, nil //nolint:nilerr // process lookup failure means not running
	}

	if sigErr := proc.Signal(syscall.Signal(0)); sigErr != nil {
		return false, pid, nil //nolint:nilerr // signal failure means process is not alive
	}

	return true, pid, nil
}

func writePIDFile() error {
	if err := os.WriteFile(opts.PIDFile(), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil { //nolint:mnd
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

func removePIDFile() {
	_ = os.Remove(opts.PIDFile())
}

func runEmbeddedZot(parentCtx context.Context, address string, rootDirectory string) context.Context {
	ctx, cancel := context.WithCancel(parentCtx)

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		logger.Error("failed to split host and port", "error", err)

		cancel()

		return ctx
	}

	conf := zotconfig.New()
	conf.Storage.RootDirectory = rootDirectory

	conf.HTTP.Address = host
	conf.HTTP.Port = port
	conf.Log.Level = "error"

	ctrl := zotapi.NewController(conf)

	go func(ctx context.Context) {
		shutdown := func() {
			defer cancel()

			logger.Info("shutting down zot controller")
			ctrl.Shutdown()
		}

		go func() {
			<-ctx.Done()
			shutdown()
		}()

		defer func() {
			shutdown()
		}()

		if err := ctrl.Init(); err != nil {
			logger.Error("failed to init zot controller", "error", err)

			return
		}

		if err := ctrl.Run(); err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Error("failed to run zot controller", "error", err)
			}

			return
		}
	}(ctx)

	return ctx
}

func isZotReady(address string) (bool, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/v2/", address)) //nolint:noctx
	if err != nil {
		logger.Debug("zot readiness check failed with error", "error", err)

		return false, fmt.Errorf("failed to check zot readiness: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Debug("zot readiness check failed with status code", "status_code", resp.StatusCode)

		return false, fmt.Errorf("zot was not ready, /v2/ returned status code: %d", resp.StatusCode)
	}

	logger.Debug("zot readiness check passed")

	return true, nil
}

// Command is the parent command for daemon subcommands.
var Command = &cobra.Command{
	Use:   "daemon",
	Short: "Run a local directory server",
	Long: `Run a self-contained local directory server that bundles the gRPC apiserver
and reconciler into a single process.

All data is stored under ~/.agntcy/dir/ by default.
Without --config, built-in defaults are used. When --config is provided the
file is read as the complete configuration (no merging with defaults).

Examples:
  # Start the daemon with built-in defaults
  dirctl daemon start

  # Start with a custom config
  dirctl daemon start --config /path/to/config.yaml

  # Stop a running daemon
  dirctl daemon stop

  # Check daemon status
  dirctl daemon status`,
	// Override root PersistentPreRunE: daemon IS the server, no client needed.
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return nil
	},
}

func init() {
	Command.PersistentFlags().StringVar(&opts.DataDir, "data-dir", defaultDataDir(), "Data directory for daemon state")
	Command.PersistentFlags().StringVar(&opts.ConfigFile, "config", "", "Path to daemon config file (default: <data-dir>/"+DefaultConfigFile+")")

	// Hide all root command flags since they are not relevant to the daemon command
	Command.SetHelpFunc(func(cmd *cobra.Command, strings []string) {
		cmd.Root().Flags().VisitAll(func(f *pflag.Flag) { f.Hidden = true })
		cmd.Print(cmd.UsageString())
	})

	// Register subcommands
	Command.AddCommand(
		startCmd,
		stopCmd,
		statusCmd,
	)
}
