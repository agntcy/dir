// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"errors"
	"fmt"
	"strings"

	routing "github.com/agntcy/dir/server/routing/config"
	localfs "github.com/agntcy/dir/server/store/localfs/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix = "DIRECTORY_SERVER"

	// API configuration.

	DefaultListenAddress      = "0.0.0.0:8888"
	DefaultHealthCheckAddress = "0.0.0.0:8889"

	// Provider configuration.

	DefaultProvider = "oci"

	DefaultConfigName = "server.config"
	DefaultConfigType = "yml"
	DefaultConfigPath = "/etc/agntcy/dir"
)

var logger = logging.Logger("config")

type Config struct {
	// API configuration
	ListenAddress      string `json:"listen_address,omitempty"      mapstructure:"listen_address"`
	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`

	// Provider configuration
	Provider string         `json:"provider,omitempty" mapstructure:"provider"`
	LocalFS  localfs.Config `json:"localfs,omitempty"  mapstructure:"localfs"`
	OCI      oci.Config     `json:"oci,omitempty"      mapstructure:"oci"`

	// Routing configuration
	Routing routing.Config `json:"routing,omitempty" mapstructure:"routing"`
}

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
			logger.Info("Config file not found, use defaults.")
		} else {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	//
	// API configuration
	//
	_ = v.BindEnv("listen_address")
	v.SetDefault("listen_address", DefaultListenAddress)

	_ = v.BindEnv("healthcheck_address")
	v.SetDefault("healthcheck_address", DefaultHealthCheckAddress)

	//
	// Provider configuration
	//
	_ = v.BindEnv("provider")
	v.SetDefault("provider", DefaultProvider)

	//
	// LocalFS configuration
	//
	_ = v.BindEnv("localfs.dir")
	v.SetDefault("localfs.dir", localfs.DefaultDir)

	//
	// OCI configuration
	//
	_ = v.BindEnv("oci.local_dir")
	v.SetDefault("oci.local_dir", "")

	_ = v.BindEnv("oci.cache_dir")
	v.SetDefault("oci.cache_dir", "")

	_ = v.BindEnv("oci.registry_address")
	v.SetDefault("oci.registry_address", oci.DefaultRegistryAddress)

	_ = v.BindEnv("oci.repository_name")
	v.SetDefault("oci.repository_name", oci.DefaultRepositoryName)

	_ = v.BindEnv("oci.auth_config.insecure")
	v.SetDefault("oci.auth_config.insecure", oci.DefaultAuthConfigInsecure)

	_ = v.BindEnv("oci.auth_config.username")
	_ = v.BindEnv("oci.auth_config.password")
	_ = v.BindEnv("oci.auth_config.access_token")
	_ = v.BindEnv("oci.auth_config.refresh_token")

	//
	// Routing configuration
	//
	_ = v.BindEnv("routing.listen_address")
	v.SetDefault("routing.listen_address", routing.DefaultListenAddress)

	_ = v.BindEnv("routing.bootstrap_peers")
	v.SetDefault("routing.bootstrap_peers", strings.Join(routing.DefaultBootstrapPeers, ","))

	_ = v.BindEnv("routing.key_path")
	v.SetDefault("routing.key_path", "")

	_ = v.BindEnv("routing.datastore_dir")
	v.SetDefault("routing.datastore_dir", "")

	// Load configuration into struct
	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	config := &Config{}
	if err := v.Unmarshal(config, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}
