// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"

	localfs "github.com/agntcy/dir/server/store/localfs/config"
	oci "github.com/agntcy/dir/server/store/oci/config"
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
)

type Config struct {
	// API configuration
	ListenAddress      string `json:"listen_address,omitempty"      mapstructure:"listen_address"`
	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`

	// Provider configuration
	Provider string         `json:"provider,omitempty" mapstructure:"provider"`
	LocalFS  localfs.Config `json:"localfs,omitempty"  mapstructure:"localfs"`
	OCI      oci.Config     `json:"oci,omitempty"      mapstructure:"oci"`
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

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
