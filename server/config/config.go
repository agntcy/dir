// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/agntcy/dir/server/store/localfs"
	"github.com/agntcy/dir/server/store/oci"
)

const (
	DefaultEnvPrefix = "DIRECTORY_SERVER"

	// API configuration

	DefaultListenAddress      = "0.0.0.0:8888"
	DefaultHealthCheckAddress = "0.0.0.0:8889"

	// Provider configuration

	DefaultProvider = "oci"
)

type Config struct {
	// API configuration
	ListenAddress      string `json:"listen_address,omitempty" mapstructure:"listen_address"`
	HealthCheckAddress string `json:"healthcheck_address,omitempty" mapstructure:"healthcheck_address"`
	// Provider configuration
	Provider string `json:"provider,omitempty" mapstructure:"provider"`
	// LocalFS configuration
	LocalFS localfs.Config `json:",inline" mapstructure:",squash"`
	// OCI configuration
	OCI oci.Config `json:",inline" mapstructure:",squash"`
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
	_ = v.BindEnv("localfs_dir")
	v.SetDefault("localfs_dir", localfs.DefaultDir)

	//
	// OCI configuration
	//
	_ = v.BindEnv("oci_registry_address")
	v.SetDefault("oci_registry_address", oci.DefaultRegistryAddress)

	_ = v.BindEnv("oci_repository_name")
	v.SetDefault("oci_repository_name", oci.DefaultRepositoryName)

	_ = v.BindEnv("oci_zot_username")
	_ = v.BindEnv("oci_zot_password")

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
