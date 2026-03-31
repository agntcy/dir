// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	reconcilerconfig "github.com/agntcy/dir/reconciler/config"
	serverconfig "github.com/agntcy/dir/server/config"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigFile is the default daemon config filename, stored under DataDir.
	DefaultConfigFile = "daemon.config.yaml"

	// DefaultEnvPrefix is the environment variable prefix for daemon configuration.
	DefaultEnvPrefix = "DIRECTORY_DAEMON"
)

// DaemonConfig is the top-level daemon configuration combining server and reconciler settings.
type DaemonConfig struct {
	Server     serverconfig.Config     `json:"server"     mapstructure:"server"`
	Reconciler reconcilerconfig.Config `json:"reconciler" mapstructure:"reconciler"`
}

// defaultConfigYAML is the single source of truth for daemon default configuration,
// embedded from daemon.config.yaml. Path fields use relative values (e.g. "store",
// "dir.db") which are resolved against opts.DataDir after loading.
//
//go:embed daemon.config.yaml
var defaultConfigYAML string

// loadConfig loads the daemon configuration. If a config file exists at the
// path derived from opts, it is merged on top of the embedded defaults.
// Otherwise the embedded defaults are used as-is.
func loadConfig() (*DaemonConfig, error) {
	cfgPath := opts.ConfigFilePath()

	cfg, err := readConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

// readConfig parses the daemon configuration. The embedded defaultConfigYAML is
// loaded as the base. If a user config file exists at path it is merged on top,
// so omitted fields retain sensible defaults. Relative paths are resolved
// against opts.DataDir after unmarshaling.
func readConfig(path string) (*DaemonConfig, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetConfigType("yaml")
	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	if err := v.ReadConfig(strings.NewReader(defaultConfigYAML)); err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	bindCredentialEnvVars(v)

	v.SetConfigFile(path)

	if err := v.MergeInConfig(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		logger.Info("No config file found, using defaults", "path", path)
	}

	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	cfg := &DaemonConfig{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	cfg.Server.Connection = cfg.Server.Connection.WithDefaults()
	resolveRelativePaths(cfg)

	return cfg, nil
}

// bindCredentialEnvVars registers credential keys that have no defaults in the
// YAML file. Without explicit BindEnv calls, AutomaticEnv cannot discover keys
// that viper has never seen. The single-argument form uses the env prefix to
// derive the env var name (e.g. DIRECTORY_DAEMON_SERVER_DATABASE_POSTGRES_USERNAME).
func bindCredentialEnvVars(v *viper.Viper) {
	_ = v.BindEnv("server.database.postgres.username")
	_ = v.BindEnv("server.database.postgres.password")

	_ = v.BindEnv("server.store.oci.auth_config.username")
	_ = v.BindEnv("server.store.oci.auth_config.password")
	_ = v.BindEnv("server.store.oci.auth_config.access_token")
	_ = v.BindEnv("server.store.oci.auth_config.refresh_token")

	_ = v.BindEnv("server.sync.auth_config.username")
	_ = v.BindEnv("server.sync.auth_config.password")
}

// resolveRelativePaths resolves path fields against opts.DataDir when they are
// relative. Absolute paths set by the user in the config file are left as-is.
func resolveRelativePaths(cfg *DaemonConfig) {
	resolve := func(p string) string {
		if filepath.IsAbs(p) {
			return p
		}

		return filepath.Join(opts.DataDir, p)
	}

	cfg.Server.Store.OCI.LocalDir = resolve(cfg.Server.Store.OCI.LocalDir)
	cfg.Server.Routing.DatastoreDir = resolve(cfg.Server.Routing.DatastoreDir)
	cfg.Server.Database.SQLite.Path = resolve(cfg.Server.Database.SQLite.Path)
}
