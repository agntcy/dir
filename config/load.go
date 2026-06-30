// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/agntcy/dir/config/auth"
	"github.com/agntcy/dir/config/naming"
	"github.com/agntcy/dir/config/reconciler"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var logger = logging.Logger("config")

// Options controls how LoadConfig discovers and reads the configuration.
type Options struct {
	file        string
	reader      io.Reader
	readerType  string
	envPrefix   string
	configName  string
	configType  string
	configPaths []string
}

// Option mutates an Options value.
type Option func(*Options)

// WithFile loads the configuration from the given file path.
func WithFile(file string) Option {
	return func(o *Options) { o.file = file }
}

// WithReader loads the configuration from the given reader.
// fileType is the viper format (yaml, json, toml). Ignored if WithFile is set.
func WithReader(r io.Reader, fileType string) Option {
	return func(o *Options) {
		o.reader = r
		o.readerType = fileType
	}
}

// WithEnvPrefix overrides the default env prefix ("DIRECTORY").
func WithEnvPrefix(prefix string) Option {
	return func(o *Options) { o.envPrefix = prefix }
}

// WithConfigName overrides the config file base name ("dir.config").
func WithConfigName(name string) Option {
	return func(o *Options) { o.configName = name }
}

// WithConfigPath appends a directory to the config file search path.
// Defaults to /etc/agntcy/dir if never called.
func WithConfigPath(path string) Option {
	return func(o *Options) { o.configPaths = append(o.configPaths, path) }
}

// LoadConfig reads the canonical dir configuration from disk and environment,
// applies defaults, and returns the populated *Config.
func LoadConfig(opts ...Option) (*Config, error) {
	options := Options{
		envPrefix:  DefaultEnvPrefix,
		configName: DefaultConfigName,
		configType: DefaultConfigType,
	}

	for _, opt := range opts {
		opt(&options)
	}

	if len(options.configPaths) == 0 {
		options.configPaths = []string{DefaultConfigPath}
	}

	v := newViper(options.envPrefix)
	registerDefaults(v)
	bindEnvKeys(v)

	if err := readViperConfig(v, options); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(defaultDecodeHooks())); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return cfg, nil
}

// LoadReconcilerConfig reads the standalone reconciler configuration from disk
// and environment, applies defaults, and returns the populated *ReconcilerConfig.
// The default env prefix is RECONCILER; override with WithEnvPrefix.
func LoadReconcilerConfig(opts ...Option) (*ReconcilerConfig, error) {
	options := Options{
		envPrefix:  DefaultReconcilerEnvPrefix,
		configName: DefaultReconcilerConfigName,
		configType: DefaultConfigType,
	}

	for _, opt := range opts {
		opt(&options)
	}

	if len(options.configPaths) == 0 {
		options.configPaths = []string{DefaultReconcilerConfigPath}
	}

	v := newViper(options.envPrefix)
	registerReconcilerProcessDefaults(v)
	bindReconcilerProcessEnvKeys(v)

	if err := readViperConfig(v, options); err != nil {
		return nil, err
	}

	cfg := &ReconcilerConfig{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(defaultDecodeHooks())); err != nil {
		return nil, fmt.Errorf("failed to load reconciler configuration: %w", err)
	}

	return cfg, nil
}

// newViper creates a viper instance with the standard key delimiter and env replacer.
func newViper(envPrefix string) *viper.Viper {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(envPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	return v
}

// readViperConfig reads config from a file, reader, or search path according to opts.
func readViperConfig(v *viper.Viper, opts Options) error {
	switch {
	case opts.file != "":
		v.SetConfigFile(opts.file)

		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("failed to read config file %q: %w", opts.file, err)
		}

	case opts.reader != nil:
		readerType := opts.readerType
		if readerType == "" {
			readerType = opts.configType
		}

		v.SetConfigType(readerType)

		if err := v.ReadConfig(opts.reader); err != nil {
			return fmt.Errorf("failed to read config from reader: %w", err)
		}

	default:
		v.SetConfigName(opts.configName)
		v.SetConfigType(opts.configType)

		for _, p := range opts.configPaths {
			v.AddConfigPath(p)
		}

		if err := v.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if errors.As(err, &notFound) {
				logger.Info("Config file not found, using defaults and environment")
			} else {
				return fmt.Errorf("failed to read configuration file: %w", err)
			}
		}
	}

	return nil
}

// defaultDecodeHooks returns the standard mapstructure decode hooks used by all loaders.
func defaultDecodeHooks() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)
}

// registerReconcilerProcessDefaults sets viper defaults for the standalone reconciler
// process. Keys are flat (no "reconciler." prefix) because the reconciler config file
// is the entire document, not a sub-section of a larger config.
func registerReconcilerProcessDefaults(v *viper.Viper) {
	v.SetDefault("database.type", DefaultDatabaseType)
	v.SetDefault("database.sqlite.path", DefaultSQLitePath())
	v.SetDefault("database.postgres.host", DefaultPostgresHost)
	v.SetDefault("database.postgres.port", DefaultPostgresPort)
	v.SetDefault("database.postgres.database", DefaultPostgresDatabase)
	v.SetDefault("database.postgres.ssl_mode", DefaultPostgresSSLMode)

	v.SetDefault("regsync.enabled", true)
	v.SetDefault("regsync.interval", reconciler.DefaultRegsyncInterval)
	v.SetDefault("regsync.timeout", reconciler.DefaultRegsyncTimeout)
	v.SetDefault("regsync.authn.enabled", false)
	v.SetDefault("regsync.authn.mode", string(auth.ModeX509))

	v.SetDefault("indexer.enabled", true)
	v.SetDefault("indexer.interval", reconciler.DefaultIndexerInterval)

	v.SetDefault("name.enabled", false)
	v.SetDefault("name.interval", reconciler.DefaultNameInterval)
	v.SetDefault("name.ttl", naming.DefaultTTL)
	v.SetDefault("name.record_timeout", reconciler.DefaultNameRecordTimeout)

	v.SetDefault("signature.enabled", true)
	v.SetDefault("signature.interval", reconciler.DefaultSignatureInterval)
	v.SetDefault("signature.ttl", reconciler.DefaultSignatureTTL)
	v.SetDefault("signature.record_timeout", reconciler.DefaultSignatureRecordTimeout)

	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.interval", reconciler.DefaultMetricsInterval)

	v.SetDefault("server_authn.enabled", false)
	v.SetDefault("server_authn.mode", string(auth.ModeX509))
}

// bindReconcilerProcessEnvKeys explicitly binds env-only keys for the standalone
// reconciler (credentials, optional paths, and fields that never appear in a config file).
func bindReconcilerProcessEnvKeys(v *viper.Viper) {
	for _, key := range []string{
		"local_registry.local_dir",
		"local_registry.cache_dir",
		"local_registry.registry_address",
		"local_registry.repository_name",
		"local_registry.auth_config.username",
		"local_registry.auth_config.password",
		"local_registry.auth_config.insecure",
		"database.postgres.username",
		"database.postgres.password",
		"regsync.authn.socket_path",
		"regsync.authn.audiences",
		"server_address",
		"server_authn.socket_path",
		"server_authn.audiences",
		"schema_url",
	} {
		_ = v.BindEnv(key)
	}
}

// registerDefaults sets viper defaults for all keys that may be absent from the
// config file and have no environment variable set. Keep in sync with Config fields.
func registerDefaults(v *viper.Viper) {
	v.SetDefault("store.registry_address", DefaultRegistryAddress)
	v.SetDefault("store.repository_name", DefaultRepositoryName)
	v.SetDefault("store.auth_config.insecure", DefaultRegistryAuthInsecure)

	v.SetDefault("database.type", DefaultDatabaseType)
	v.SetDefault("database.sqlite.path", DefaultSQLitePath())
	v.SetDefault("database.postgres.host", DefaultPostgresHost)
	v.SetDefault("database.postgres.port", DefaultPostgresPort)
	v.SetDefault("database.postgres.database", DefaultPostgresDatabase)
	v.SetDefault("database.postgres.ssl_mode", DefaultPostgresSSLMode)

	v.SetDefault("logging.verbose", false)

	v.SetDefault("server.listen_address", DefaultListenAddress)
	v.SetDefault("server.metrics.enabled", DefaultMetricsEnabled)
	v.SetDefault("server.metrics.address", DefaultMetricsAddress)

	v.SetDefault("server.ratelimit.enabled", false)
	v.SetDefault("server.ratelimit.global_rps", 0.0)
	v.SetDefault("server.ratelimit.global_burst", 0)
	v.SetDefault("server.ratelimit.per_client_rps", 0.0)
	v.SetDefault("server.ratelimit.per_client_burst", 0)

	v.SetDefault("server.authn.enabled", false)
	v.SetDefault("server.authn.mode", string(auth.ModeX509))
	v.SetDefault("server.authn.socket_path", "")
	v.SetDefault("server.authn.audiences", "")

	v.SetDefault("server.authz.enabled", false)
	v.SetDefault("server.authz.enforcer_policy_file_path", DefaultConfigPath+"/authz_policies.csv")

	v.SetDefault("server.routing.listen_address", DefaultRoutingListenAddress)
	v.SetDefault("server.routing.directory_api_address", "")
	v.SetDefault("server.routing.bootstrap_peers", defaultBootstrapPeersStr())
	v.SetDefault("server.routing.key_path", "")
	v.SetDefault("server.routing.datastore_dir", "")
	v.SetDefault("server.routing.gossipsub.enabled", DefaultGossipSubEnabled)

	v.SetDefault("server.publication.scheduler_interval", DefaultPublicationSchedulerInterval)
	v.SetDefault("server.publication.worker_count", DefaultPublicationWorkerCount)
	v.SetDefault("server.publication.worker_timeout", DefaultPublicationWorkerTimeout)

	v.SetDefault("server.events.subscriber_buffer_size", DefaultEventsSubscriberBufferSize)
	v.SetDefault("server.events.log_slow_consumers", DefaultEventsLogSlowConsumers)
	v.SetDefault("server.events.log_published_events", DefaultEventsLogPublishedEvents)

	v.SetDefault("server.naming.enabled", naming.DefaultEnabled)
	v.SetDefault("server.naming.ttl", naming.DefaultTTL)

	v.SetDefault("server.http_gateway.enabled", DefaultHTTPGatewayEnabled)
	v.SetDefault("server.http_gateway.listen_address", DefaultHTTPGatewayAddress)
	v.SetDefault("server.http_gateway.public_url", DefaultHTTPGatewayPublicURL)
	v.SetDefault("server.http_gateway.catalog_title", DefaultHTTPGatewayCatalogTitle)

	v.SetDefault("reconciler.regsync.enabled", true)
	v.SetDefault("reconciler.regsync.interval", reconciler.DefaultRegsyncInterval)
	v.SetDefault("reconciler.regsync.timeout", reconciler.DefaultRegsyncTimeout)
	v.SetDefault("reconciler.regsync.authn.enabled", false)
	v.SetDefault("reconciler.regsync.authn.mode", string(auth.ModeX509))

	v.SetDefault("reconciler.indexer.enabled", true)
	v.SetDefault("reconciler.indexer.interval", reconciler.DefaultIndexerInterval)

	v.SetDefault("reconciler.name.enabled", false)
	v.SetDefault("reconciler.name.interval", reconciler.DefaultNameInterval)
	v.SetDefault("reconciler.name.ttl", naming.DefaultTTL)
	v.SetDefault("reconciler.name.record_timeout", reconciler.DefaultNameRecordTimeout)

	v.SetDefault("reconciler.signature.enabled", true)
	v.SetDefault("reconciler.signature.interval", reconciler.DefaultSignatureInterval)
	v.SetDefault("reconciler.signature.ttl", reconciler.DefaultSignatureTTL)
	v.SetDefault("reconciler.signature.record_timeout", reconciler.DefaultSignatureRecordTimeout)

	v.SetDefault("reconciler.metrics.enabled", true)
	v.SetDefault("reconciler.metrics.interval", reconciler.DefaultMetricsInterval)
}

// bindEnvKeys explicitly binds keys that may never appear in the config file
// (credentials, optional URLs). Without explicit BindEnv calls, AutomaticEnv
// cannot discover them when no default is registered.
func bindEnvKeys(v *viper.Viper) {
	for _, key := range []string{
		"oasf_api_validation.schema_url",

		"store.local_dir",
		"store.cache_dir",
		"store.auth_config.username",
		"store.auth_config.password",
		"store.auth_config.access_token",
		"store.auth_config.refresh_token",

		"database.postgres.username",
		"database.postgres.password",

		"server.sync.auth_config.username",
		"server.sync.auth_config.password",

		"reconciler.regsync.authn.socket_path",
		"reconciler.regsync.authn.audiences",
	} {
		_ = v.BindEnv(key)
	}
}
