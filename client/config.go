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

	DefaultServerAddress         = "0.0.0.0:8888"
	DefaultSpiffeWorkloadAddress = "tcp://0.0.0.0:8081"
)

var DefaultConfig = Config{
	ServerAddress:         DefaultServerAddress,
	SpiffeWorkloadAddress: DefaultSpiffeWorkloadAddress,
}

type Config struct {
	ServerAddress         string `json:"server_address,omitempty" mapstructure:"server_address"`
	SpiffeWorkloadAddress string `json:"spiffe_workload_address,omitempty" mapstructure:"spiffe_workload_address"`
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

	_ = v.BindEnv("spiffe_workload_address")
	v.SetDefault("spiffe_workload_address", DefaultSpiffeWorkloadAddress)

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
