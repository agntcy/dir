// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config handles configuration loading for the reconciler service.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/reconciler/tasks/indexer"
	"github.com/agntcy/dir/reconciler/tasks/name"
	"github.com/agntcy/dir/reconciler/tasks/regsync"
	dbconfig "github.com/agntcy/dir/server/database/config"
	namingconfig "github.com/agntcy/dir/server/naming/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/spf13/viper"
)

const (
	// DefaultEnvPrefix is the environment variable prefix.
	DefaultEnvPrefix = "RECONCILER"

	// DefaultConfigName is the default configuration file name.
	DefaultConfigName = "reconciler.config"

	// DefaultConfigType is the default configuration file type.
	DefaultConfigType = "yml"

	// DefaultConfigPath is the default configuration file path.
	DefaultConfigPath = "/etc/agntcy/reconciler"
)

var logger = logging.Logger("reconciler/config")

// Config holds the reconciler configuration.
type Config struct {
	// Database holds PostgreSQL connection configuration.
	Database dbconfig.Config `json:"database" mapstructure:"database"`

	// LocalRegistry holds configuration for the local OCI registry.
	LocalRegistry ociconfig.Config `json:"local_registry" mapstructure:"local_registry"`

	// SchemaURL is the OASF schema URL for record validation.
	SchemaURL string `json:"schema_url" mapstructure:"schema_url"`

	// Regsync holds the regsync task configuration.
	Regsync regsync.Config `json:"regsync" mapstructure:"regsync"`

	// Indexer holds the indexer task configuration.
	Indexer indexer.Config `json:"indexer" mapstructure:"indexer"`

	// Name holds the name (name/DNS verification) task configuration.
	Name name.Config `json:"name" mapstructure:"name"`
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

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		fileNotFoundError := viper.ConfigFileNotFoundError{}
		if errors.As(err, &fileNotFoundError) {
			logger.Info("Config file not found, using defaults")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// Database configuration
	//
	_ = v.BindEnv("database.type")
	v.SetDefault("database.type", dbconfig.DefaultType)

	// PostgreSQL configuration
	_ = v.BindEnv("database.postgres.host")
	v.SetDefault("database.postgres.host", dbconfig.DefaultPostgresHost)

	_ = v.BindEnv("database.postgres.port")
	v.SetDefault("database.postgres.port", dbconfig.DefaultPostgresPort)

	_ = v.BindEnv("database.postgres.database")
	v.SetDefault("database.postgres.database", dbconfig.DefaultPostgresDatabase)

	_ = v.BindEnv("database.postgres.username")
	_ = v.BindEnv("database.postgres.password")

	_ = v.BindEnv("database.postgres.ssl_mode")
	v.SetDefault("database.postgres.ssl_mode", dbconfig.DefaultPostgresSSLMode)

	//
	// Local registry configuration (shared by all tasks)
	//
	_ = v.BindEnv("local_registry.registry_address")
	_ = v.BindEnv("local_registry.repository_name")
	_ = v.BindEnv("local_registry.auth_config.username")
	_ = v.BindEnv("local_registry.auth_config.password")
	_ = v.BindEnv("local_registry.auth_config.insecure")

	//
	// Regsync task configuration
	//
	_ = v.BindEnv("regsync.enabled")
	v.SetDefault("regsync.enabled", true)

	_ = v.BindEnv("regsync.interval")
	v.SetDefault("regsync.interval", regsync.DefaultInterval)

	_ = v.BindEnv("regsync.binary_path")
	v.SetDefault("regsync.binary_path", regsync.DefaultBinaryPath)

	_ = v.BindEnv("regsync.timeout")
	v.SetDefault("regsync.timeout", regsync.DefaultTimeout)

	//
	// Authentication configuration for registry credentials provider
	//
	_ = v.BindEnv("regsync.authn.enabled")
	v.SetDefault("regsync.authn.enabled", false)

	_ = v.BindEnv("regsync.authn.mode")
	v.SetDefault("regsync.authn.mode", "x509")

	_ = v.BindEnv("regsync.authn.socket_path")
	_ = v.BindEnv("regsync.authn.audiences")

	//
	// Indexer task configuration
	//
	_ = v.BindEnv("indexer.enabled")
	v.SetDefault("indexer.enabled", true)

	_ = v.BindEnv("indexer.interval")
	v.SetDefault("indexer.interval", indexer.DefaultInterval)

	//
	// Name task configuration (name/DNS verification)
	//
	_ = v.BindEnv("name.enabled")
	v.SetDefault("name.enabled", false)

	_ = v.BindEnv("name.interval")
	v.SetDefault("name.interval", name.DefaultInterval)

	_ = v.BindEnv("name.ttl")
	v.SetDefault("name.ttl", namingconfig.DefaultTTL)

	_ = v.BindEnv("name.record_timeout")
	v.SetDefault("name.record_timeout", name.DefaultRecordTimeout)

	//
	// OASF validation configuration
	//
	_ = v.BindEnv("schema_url")

	// Unmarshal into config struct
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	return config, nil
}
