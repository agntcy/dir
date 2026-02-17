// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config handles configuration loading for the reconciler service.
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/agntcy/dir/reconciler/tasks/indexer"
	"github.com/agntcy/dir/reconciler/tasks/regsync"
	dbconfig "github.com/agntcy/dir/server/database/config"
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
	_ = v.BindEnv("database.host")
	v.SetDefault("database.host", dbconfig.DefaultPostgresHost)

	_ = v.BindEnv("database.port")
	v.SetDefault("database.port", dbconfig.DefaultPostgresPort)

	_ = v.BindEnv("database.database")
	v.SetDefault("database.database", dbconfig.DefaultPostgresDatabase)

	_ = v.BindEnv("database.username")
	_ = v.BindEnv("database.password")

	_ = v.BindEnv("database.ssl_mode")
	v.SetDefault("database.ssl_mode", dbconfig.DefaultPostgresSSLMode)

	//
	// Local registry configuration (shared by all tasks)
	//
	_ = v.BindEnv("local_registry.type")
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

	_ = v.BindEnv("regsync.config_path")
	v.SetDefault("regsync.config_path", regsync.DefaultConfigPath)

	_ = v.BindEnv("regsync.binary_path")
	v.SetDefault("regsync.binary_path", regsync.DefaultBinaryPath)

	_ = v.BindEnv("regsync.timeout")
	v.SetDefault("regsync.timeout", regsync.DefaultTimeout)

	// Authentication configuration for remote Directory connections
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
