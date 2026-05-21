// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config handles configuration loading for the reconciler
// service. The actual schema (Database, Registry, task settings) lives
// in github.com/agntcy/dir/config; this package only wires viper for
// the standalone reconciler binary.
package config

import (
	"errors"
	"fmt"
	"strings"

	dircfg "github.com/agntcy/dir/config"
	namingcfg "github.com/agntcy/dir/config/naming"
	reconcilercfg "github.com/agntcy/dir/config/reconciler"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/viper"
)

const (
	// DefaultEnvPrefix is the environment variable prefix for the
	// standalone reconciler binary.
	DefaultEnvPrefix = "RECONCILER"

	// DefaultConfigName is the default configuration file name.
	DefaultConfigName = "reconciler.config"

	// DefaultConfigType is the default configuration file type.
	DefaultConfigType = "yml"

	// DefaultConfigPath is the default configuration file path.
	DefaultConfigPath = "/etc/agntcy/reconciler"
)

var logger = logging.Logger("reconciler/config")

// Config holds the reconciler configuration. Each field aliases the
// canonical type from github.com/agntcy/dir/config so that the
// reconciler binary and the in-process daemon both ingest the same
// schema.
type Config struct {
	// Database holds the shared state-store configuration.
	Database dircfg.Database `json:"database" mapstructure:"database"`

	// LocalRegistry holds the OCI registry both apiserver and
	// reconciler read from. Named "local_registry" for backwards
	// compatibility with existing reconciler.config.yml; the canonical
	// Config exposes it as "registry".
	LocalRegistry dircfg.Registry `json:"local_registry" mapstructure:"local_registry"`

	// SchemaURL is the OASF schema URL for record validation.
	SchemaURL string `json:"schema_url" mapstructure:"schema_url"`

	// Regsync holds the regsync task configuration.
	Regsync reconcilercfg.Regsync `json:"regsync" mapstructure:"regsync"`

	// Indexer holds the indexer task configuration.
	Indexer reconcilercfg.Indexer `json:"indexer" mapstructure:"indexer"`

	// Name holds the name (DNS / well-known) verification task configuration.
	Name reconcilercfg.Name `json:"name" mapstructure:"name"`

	// Signature holds the signature verification task configuration.
	Signature reconcilercfg.Signature `json:"signature" mapstructure:"signature"`
}

// LoadConfig loads the configuration from file and environment variables.
func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetConfigName(DefaultConfigName)
	v.SetConfigType(DefaultConfigType)
	v.AddConfigPath(DefaultConfigPath)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if errors.As(err, &notFound) {
			logger.Info("Config file not found, using defaults")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	// Database configuration.
	_ = v.BindEnv("database.type")
	v.SetDefault("database.type", dircfg.DefaultDatabaseType)

	_ = v.BindEnv("database.sqlite.path")
	v.SetDefault("database.sqlite.path", dircfg.DefaultSQLitePath)

	_ = v.BindEnv("database.postgres.host")
	v.SetDefault("database.postgres.host", dircfg.DefaultPostgresHost)

	_ = v.BindEnv("database.postgres.port")
	v.SetDefault("database.postgres.port", dircfg.DefaultPostgresPort)

	_ = v.BindEnv("database.postgres.database")
	v.SetDefault("database.postgres.database", dircfg.DefaultPostgresDatabase)

	_ = v.BindEnv("database.postgres.username")
	_ = v.BindEnv("database.postgres.password")

	_ = v.BindEnv("database.postgres.ssl_mode")
	v.SetDefault("database.postgres.ssl_mode", dircfg.DefaultPostgresSSLMode)

	// Local registry configuration (shared by all tasks).
	_ = v.BindEnv("local_registry.registry_address")
	_ = v.BindEnv("local_registry.repository_name")
	_ = v.BindEnv("local_registry.auth_config.username")
	_ = v.BindEnv("local_registry.auth_config.password")
	_ = v.BindEnv("local_registry.auth_config.insecure")

	// Regsync task configuration.
	_ = v.BindEnv("regsync.enabled")
	v.SetDefault("regsync.enabled", true)

	_ = v.BindEnv("regsync.interval")
	v.SetDefault("regsync.interval", reconcilercfg.DefaultRegsyncInterval)

	_ = v.BindEnv("regsync.timeout")
	v.SetDefault("regsync.timeout", reconcilercfg.DefaultRegsyncTimeout)

	// Authentication for the registry credentials provider.
	_ = v.BindEnv("regsync.authn.enabled")
	v.SetDefault("regsync.authn.enabled", false)

	_ = v.BindEnv("regsync.authn.mode")
	v.SetDefault("regsync.authn.mode", "x509")

	_ = v.BindEnv("regsync.authn.socket_path")
	_ = v.BindEnv("regsync.authn.audiences")

	// Indexer task configuration.
	_ = v.BindEnv("indexer.enabled")
	v.SetDefault("indexer.enabled", true)

	_ = v.BindEnv("indexer.interval")
	v.SetDefault("indexer.interval", reconcilercfg.DefaultIndexerInterval)

	// Name task configuration (DNS / well-known verification).
	_ = v.BindEnv("name.enabled")
	v.SetDefault("name.enabled", false)

	_ = v.BindEnv("name.interval")
	v.SetDefault("name.interval", reconcilercfg.DefaultNameInterval)

	_ = v.BindEnv("name.ttl")
	v.SetDefault("name.ttl", namingcfg.DefaultTTL)

	_ = v.BindEnv("name.record_timeout")
	v.SetDefault("name.record_timeout", reconcilercfg.DefaultNameRecordTimeout)

	// Signature task configuration.
	_ = v.BindEnv("signature.enabled")
	v.SetDefault("signature.enabled", true)

	_ = v.BindEnv("signature.interval")
	v.SetDefault("signature.interval", reconcilercfg.DefaultSignatureInterval)

	_ = v.BindEnv("signature.ttl")
	v.SetDefault("signature.ttl", reconcilercfg.DefaultSignatureTTL)

	_ = v.BindEnv("signature.record_timeout")
	v.SetDefault("signature.record_timeout", reconcilercfg.DefaultSignatureRecordTimeout)

	// OASF validation configuration.
	_ = v.BindEnv("schema_url")

	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return cfg, nil
}
