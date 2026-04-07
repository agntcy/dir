// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	_ "embed"
	"fmt"
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

//go:embed daemon.config.yaml
var defaultConfigYAML string

// loadConfig loads the daemon configuration. When the user provides a config
// file via --config, that file is read as-is (no defaults merged). Otherwise
// the embedded daemon.config.yaml is used as the complete default configuration.
func loadConfig() (*DaemonConfig, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetConfigType("yaml")
	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	bindCredentialEnvVars(v)

	if opts.ConfigFile != "" {
		v.SetConfigFile(opts.ConfigFile)

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		if err := v.ReadConfig(strings.NewReader(defaultConfigYAML)); err != nil {
			return nil, fmt.Errorf("failed to load embedded default config: %w", err)
		}
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

// bindCredentialEnvVars registers credential keys so that AutomaticEnv can
// resolve them. Without explicit BindEnv calls, viper cannot discover keys
// that never appear in a config file.
func bindCredentialEnvVars(v *viper.Viper) {
	_ = v.BindEnv("server.database.postgres.username")
	_ = v.BindEnv("server.database.postgres.password")

	_ = v.BindEnv("server.routing.bootstrap_peers")

	_ = v.BindEnv("server.store.oci.auth_config.username")
	_ = v.BindEnv("server.store.oci.auth_config.password")
	_ = v.BindEnv("server.store.oci.auth_config.access_token")
	_ = v.BindEnv("server.store.oci.auth_config.refresh_token")

	_ = v.BindEnv("server.sync.auth_config.username")
	_ = v.BindEnv("server.sync.auth_config.password")
}

// resolveRelativePaths resolves non-empty path fields against opts.DataDir
// when they are relative. Empty paths are left for the service to default.
// Absolute paths set by the user are left as-is.
func resolveRelativePaths(cfg *DaemonConfig) {
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}

		return filepath.Join(opts.DataDir, p)
	}

	cfg.Server.Store.OCI.LocalDir = resolve(cfg.Server.Store.OCI.LocalDir)
	cfg.Server.Routing.KeyPath = resolve(cfg.Server.Routing.KeyPath)
	cfg.Server.Routing.DatastoreDir = resolve(cfg.Server.Routing.DatastoreDir)
	cfg.Server.Database.SQLite.Path = resolve(cfg.Server.Database.SQLite.Path)
}
