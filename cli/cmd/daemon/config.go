// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	resolvera2a "github.com/agntcy/dir-runtime/discovery/resolver/a2a"
	resolver "github.com/agntcy/dir-runtime/discovery/resolver/config"
	resolveroasf "github.com/agntcy/dir-runtime/discovery/resolver/oasf"
	runtime "github.com/agntcy/dir-runtime/discovery/runtime/config"
	adapterdocker "github.com/agntcy/dir-runtime/discovery/runtime/docker"
	adapterk8s "github.com/agntcy/dir-runtime/discovery/runtime/k8s"
	runtimestore "github.com/agntcy/dir-runtime/store/config"
	runtimestoresql "github.com/agntcy/dir-runtime/store/sql"
	dircfg "github.com/agntcy/dir/config"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	// DefaultConfigFile is the default daemon config filename, stored under DataDir.
	DefaultConfigFile = "daemon.config.yaml"

	// DefaultEnvPrefix is the environment variable prefix for daemon configuration.
	DefaultEnvPrefix = "DIRECTORY_DAEMON"
)

// DaemonConfig is the top-level daemon configuration.
// Shared infrastructure (Store, Database) and apiserver settings are at the top level;
// reconciler-specific and runtime settings are nested under their own keys.
type DaemonConfig struct {
	// Shared OCI store used by both the apiserver and the reconciler.
	Store dircfg.Registry `json:"store" mapstructure:"store"`

	// Shared database used by both the apiserver and the reconciler.
	Database dircfg.Database `json:"database" mapstructure:"database"`

	// OASFAPIValidation holds the schema URL for OASF record validation.
	OASFAPIValidation dircfg.OASFAPIValidation `json:"oasf_api_validation" mapstructure:"oasf_api_validation"`

	// Logging configures process-wide logging.
	Logging dircfg.Logging `json:"logging" mapstructure:"logging"`

	// APIServer holds settings specific to the gRPC apiserver.
	APIServer dircfg.APIServer `json:"server" mapstructure:"server"`

	// Reconciler holds the reconciler service configuration.
	Reconciler dircfg.ReconcilerConfig `json:"reconciler" mapstructure:"reconciler"`

	// Runtime holds configuration for the runtime adapter and resolver.
	Runtime RuntimeConfig `json:"runtime" mapstructure:"runtime"`
}

// serverConfig builds a canonical *dircfg.Config from the flat daemon config
// for passing to server.New.
func (d *DaemonConfig) serverConfig() *dircfg.Config {
	return &dircfg.Config{
		Store:             d.Store,
		Database:          d.Database,
		OASFAPIValidation: d.OASFAPIValidation,
		Logging:           d.Logging,
		APIServer:         d.APIServer,
	}
}

// RuntimeConfig holds configuration for the runtime adapter and resolver used by
// the discovery and runtime server services.
type RuntimeConfig struct {
	Enabled  bool                `json:"enabled"  mapstructure:"enabled"`
	Adapter  runtime.Config      `json:"adapter"  mapstructure:"adapter"`
	Resolver resolver.Config     `json:"resolver" mapstructure:"resolver"`
	Store    runtimestore.Config `json:"store"    mapstructure:"store"`
}

func registerServerDefaults(v *viper.Viper) {
	v.SetDefault("store.registry_address", dircfg.DefaultRegistryAddress)
	v.SetDefault("store.repository_name", dircfg.DefaultRepositoryName)
}

func registerReconcilerDefaults(v *viper.Viper) {
	v.SetDefault("reconciler.local_registry.registry_address", dircfg.DefaultRegistryAddress)
	v.SetDefault("reconciler.local_registry.repository_name", dircfg.DefaultRepositoryName)
	v.SetDefault("reconciler.local_registry.auth_config.insecure", true)
	v.SetDefault("reconciler.database.type", "sqlite")
	v.SetDefault("reconciler.database.sqlite.path", dircfg.DefaultSQLitePath())
}

func registerRuntimeDefaults(v *viper.Viper) {
	v.SetDefault("runtime.enabled", false)

	// Store configuration
	v.SetDefault("runtime.store.type", runtimestoresql.StoreTypeSqlite)

	// Adapter configuration
	v.SetDefault("runtime.adapter.type", adapterdocker.RuntimeType)

	// Docker adapter
	// Docker host mode is required for the adapter to access the Docker socket when the daemon runs in a process.
	// Users should disable host mode if they run the daemon as a container and ensure socket/networking access is properly configured.
	v.SetDefault("runtime.adapter.docker.host", adapterdocker.DefaultHost)
	v.SetDefault("runtime.adapter.docker.host_mode", true)
	v.SetDefault("runtime.adapter.docker.label_key", adapterdocker.DefaultLabelKey)
	v.SetDefault("runtime.adapter.docker.label_value", adapterdocker.DefaultLabelValue)

	// K8s adapter
	v.SetDefault("runtime.adapter.kubernetes.kubeconfig", "")
	v.SetDefault("runtime.adapter.kubernetes.namespace", adapterk8s.DefaultNamespace)
	v.SetDefault("runtime.adapter.kubernetes.label_key", adapterk8s.DefaultLabelKey)
	v.SetDefault("runtime.adapter.kubernetes.label_value", adapterk8s.DefaultLabelValue)

	// A2A resolver
	v.SetDefault("runtime.resolver.a2a.enabled", true)
	v.SetDefault("runtime.resolver.a2a.timeout", resolvera2a.DefaultTimeout)
	v.SetDefault("runtime.resolver.a2a.paths", resolvera2a.DefaultDiscoveryPaths)
	v.SetDefault("runtime.resolver.a2a.label_key", resolvera2a.DefaultLabelKey)
	v.SetDefault("runtime.resolver.a2a.label_value", resolvera2a.DefaultLabelValue)

	// OASF resolver
	v.SetDefault("runtime.resolver.oasf.enabled", true)
	v.SetDefault("runtime.resolver.oasf.timeout", resolveroasf.DefaultTimeout)
	v.SetDefault("runtime.resolver.oasf.label_key", resolveroasf.DefaultLabelKey)
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
	registerRuntimeDefaults(v)
	registerServerDefaults(v)
	registerReconcilerDefaults(v)

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

	cfg.APIServer.Connection = cfg.APIServer.Connection.WithDefaults()
	resolveRelativePaths(cfg)

	return cfg, nil
}

// bindCredentialEnvVars registers credential keys so that AutomaticEnv can
// resolve them. Without explicit BindEnv calls, viper cannot discover keys
// that never appear in a config file.
func bindCredentialEnvVars(v *viper.Viper) {
	_ = v.BindEnv("database.postgres.username")
	_ = v.BindEnv("database.postgres.password")

	_ = v.BindEnv("server.routing.bootstrap_peers")

	_ = v.BindEnv("store.auth_config.username")
	_ = v.BindEnv("store.auth_config.password")
	_ = v.BindEnv("store.auth_config.access_token")
	_ = v.BindEnv("store.auth_config.refresh_token")

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

	cfg.Store.LocalDir = resolve(cfg.Store.LocalDir)
	cfg.APIServer.Routing.KeyPath = resolve(cfg.APIServer.Routing.KeyPath)
	cfg.APIServer.Routing.DatastoreDir = resolve(cfg.APIServer.Routing.DatastoreDir)
	cfg.Database.SQLite.Path = resolve(cfg.Database.SQLite.Path)
}
