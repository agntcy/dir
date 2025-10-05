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

	DefaultServerAddress = "0.0.0.0:8888"
)

var DefaultConfig = Config{
	ServerAddress: DefaultServerAddress,
}

type Config struct {
	ServerAddress    string `json:"server_address,omitempty"     mapstructure:"server_address"`
	SpiffeSocketPath string `json:"spiffe_socket_path,omitempty" mapstructure:"spiffe_socket_path"`
	AuthMode         string `json:"auth_mode,omitempty"          mapstructure:"auth_mode"`
	JWTAudience      string `json:"jwt_audience,omitempty"       mapstructure:"jwt_audience"`
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

	_ = v.BindEnv("spiffe_socket_path")
	v.SetDefault("spiffe_socket_path", "")

	_ = v.BindEnv("auth_mode")
	v.SetDefault("auth_mode", "")

	_ = v.BindEnv("jwt_audience")
	v.SetDefault("jwt_audience", "")

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
