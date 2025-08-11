// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix = "DIRECTORY_CLIENT"

	DefaultServerAddress   = "0.0.0.0:8888"
	DefaultRegistryAddress = "127.0.0.1:5000"
	DefaultRepositoryName  = "dir"
)

var DefaultConfig = Config{
	ServerAddress:   DefaultServerAddress,
	RegistryAddress: DefaultRegistryAddress,
	RepositoryName:  DefaultRepositoryName,
}

type Config struct {
	ServerAddress     string `json:"server_address,omitempty"      mapstructure:"server_address"`
	SpiffeSocketPath  string `json:"spiffe_socket_path,omitempty"  mapstructure:"spiffe_socket_path"`
	SpiffeTrustDomain string `json:"spiffe_trust_domain,omitempty" mapstructure:"spiffe_trust_domain"`
	RegistryAddress   string `json:"registry_address,omitempty"    mapstructure:"registry_address"`
	RepositoryName    string `json:"repository_name,omitempty"     mapstructure:"repository_name"`
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	_ = v.BindEnv("server_address")
	v.SetDefault("server_address", DefaultServerAddress)

	// SPIFFE Workload API configuration
	_ = v.BindEnv("spiffe_socket_path")
	v.SetDefault("spiffe_socket_path", "")

	_ = v.BindEnv("spiffe_trust_domain")
	v.SetDefault("spiffe_trust_domain", "")

	_ = v.BindEnv("registry_address")
	v.SetDefault("registry_address", DefaultRegistryAddress)

	_ = v.BindEnv("repository_name")
	v.SetDefault("repository_name", DefaultRepositoryName)

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
