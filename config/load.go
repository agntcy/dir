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

// Options controls how LoadConfig discovers and reads the
// configuration. Use the With* helpers; do not construct Options
// directly.
type Options struct {
	file        string
	reader      io.Reader
	readerType  string
	envPrefix   string
	configName  string
	configType  string
	configPaths []string
}

// Option mutates an Options value. See WithFile, WithReader,
// WithEnvPrefix, WithConfigName, and WithConfigPath.
type Option func(*Options)

// WithFile loads the configuration from the given file path. When set,
// WithReader / WithConfigName / WithConfigPath are ignored.
func WithFile(file string) Option {
	return func(o *Options) { o.file = file }
}

// WithReader loads the configuration from the given reader. fileType
// is the viper file type (yaml, json, toml). Useful for embedded
// configs and tests. Ignored if WithFile is also passed.
func WithReader(r io.Reader, fileType string) Option {
	return func(o *Options) {
		o.reader = r
		o.readerType = fileType
	}
}

// WithEnvPrefix overrides DefaultEnvPrefix ("DIRECTORY").
func WithEnvPrefix(prefix string) Option {
	return func(o *Options) { o.envPrefix = prefix }
}

// WithConfigName overrides DefaultConfigName ("dir.config").
func WithConfigName(name string) Option {
	return func(o *Options) { o.configName = name }
}

// WithConfigPath appends a directory to the config file lookup path.
// May be called multiple times. If never called, DefaultConfigPath
// ("/etc/agntcy/dir") is used.
func WithConfigPath(path string) Option {
	return func(o *Options) { o.configPaths = append(o.configPaths, path) }
}

// LoadConfig reads the canonical dir configuration from disk and
// environment, applies defaults, and returns the populated *Config.
// With no options it looks for /etc/agntcy/dir/dir.config.yml and the
// DIRECTORY_* environment variables.
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

	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(options.envPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	registerDefaults(v)
	bindEnvKeys(v)

	switch {
	case options.file != "":
		v.SetConfigFile(options.file)

		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file %q: %w", options.file, err)
		}
	case options.reader != nil:
		readerType := options.readerType
		if readerType == "" {
			readerType = options.configType
		}

		v.SetConfigType(readerType)

		if err := v.ReadConfig(options.reader); err != nil {
			return nil, fmt.Errorf("failed to read config from reader: %w", err)
		}
	default:
		v.SetConfigName(options.configName)
		v.SetConfigType(options.configType)

		for _, p := range options.configPaths {
			v.AddConfigPath(p)
		}

		if err := v.ReadInConfig(); err != nil {
			var notFound viper.ConfigFileNotFoundError
			if errors.As(err, &notFound) {
				logger.Info("Config file not found, using defaults and environment")
			} else {
				return nil, fmt.Errorf("failed to read configuration file: %w", err)
			}
		}
	}

	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	cfg := &Config{}
	if err := v.Unmarshal(cfg, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	cfg.APIServer.Connection = cfg.APIServer.Connection.WithDefaults()

	return cfg, nil
}

// registerDefaults wires every default value for keys that may be
// missing from the config file and have no environment variable set.
// Keep this in sync with the field documentation on Config.
func registerDefaults(v *viper.Viper) {
	v.SetDefault("registry.registry_address", DefaultRegistryAddress)
	v.SetDefault("registry.repository_name", DefaultRepositoryName)
	v.SetDefault("registry.auth_config.insecure", DefaultRegistryAuthInsecure)

	v.SetDefault("database.type", DefaultDatabaseType)
	v.SetDefault("database.sqlite.path", DefaultSQLitePath)
	v.SetDefault("database.postgres.host", DefaultPostgresHost)
	v.SetDefault("database.postgres.port", DefaultPostgresPort)
	v.SetDefault("database.postgres.database", DefaultPostgresDatabase)
	v.SetDefault("database.postgres.ssl_mode", DefaultPostgresSSLMode)

	v.SetDefault("logging.verbose", false)

	v.SetDefault("apiserver.listen_address", DefaultListenAddress)
	v.SetDefault("apiserver.metrics.enabled", DefaultMetricsEnabled)
	v.SetDefault("apiserver.metrics.address", DefaultMetricsAddress)

	v.SetDefault("apiserver.ratelimit.enabled", false)
	v.SetDefault("apiserver.ratelimit.global_rps", 0.0)
	v.SetDefault("apiserver.ratelimit.global_burst", 0)
	v.SetDefault("apiserver.ratelimit.per_client_rps", 0.0)
	v.SetDefault("apiserver.ratelimit.per_client_burst", 0)

	v.SetDefault("apiserver.authn.enabled", false)
	v.SetDefault("apiserver.authn.mode", string(auth.ModeX509))
	v.SetDefault("apiserver.authn.socket_path", "")
	v.SetDefault("apiserver.authn.audiences", "")

	v.SetDefault("apiserver.authz.enabled", false)
	v.SetDefault("apiserver.authz.enforcer_policy_file_path", DefaultConfigPath+"/authz_policies.csv")

	v.SetDefault("apiserver.routing.listen_address", DefaultRoutingListenAddress)
	v.SetDefault("apiserver.routing.directory_api_address", "")
	v.SetDefault("apiserver.routing.bootstrap_peers", strings.Join(DefaultBootstrapPeers, ","))
	v.SetDefault("apiserver.routing.key_path", "")
	v.SetDefault("apiserver.routing.datastore_dir", "")
	v.SetDefault("apiserver.routing.gossipsub.enabled", DefaultGossipSubEnabled)

	v.SetDefault("apiserver.publication.scheduler_interval", DefaultPublicationSchedulerInterval)
	v.SetDefault("apiserver.publication.worker_count", DefaultPublicationWorkerCount)
	v.SetDefault("apiserver.publication.worker_timeout", DefaultPublicationWorkerTimeout)

	v.SetDefault("apiserver.events.subscriber_buffer_size", DefaultEventsSubscriberBufferSize)
	v.SetDefault("apiserver.events.log_slow_consumers", DefaultEventsLogSlowConsumers)
	v.SetDefault("apiserver.events.log_published_events", DefaultEventsLogPublishedEvents)

	v.SetDefault("apiserver.naming.enabled", naming.DefaultEnabled)
	v.SetDefault("apiserver.naming.ttl", naming.DefaultTTL)

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
}

// bindEnvKeys explicitly binds keys that may never appear in the
// config file (credentials, optional schema URL). Without explicit
// binds, viper.AutomaticEnv cannot discover them.
func bindEnvKeys(v *viper.Viper) {
	for _, key := range []string{
		"oasf_api_validation.schema_url",

		"registry.local_dir",
		"registry.cache_dir",
		"registry.auth_config.username",
		"registry.auth_config.password",
		"registry.auth_config.access_token",
		"registry.auth_config.refresh_token",

		"database.postgres.username",
		"database.postgres.password",

		"apiserver.sync.auth_config.username",
		"apiserver.sync.auth_config.password",

		"reconciler.regsync.authn.socket_path",
		"reconciler.regsync.authn.audiences",
	} {
		_ = v.BindEnv(key)
	}
}
