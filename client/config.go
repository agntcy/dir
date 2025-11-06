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
	DefaultTlsSkipVerify = false
)

var DefaultConfig = Config{
	ServerAddress: DefaultServerAddress,
}

type Config struct {
	ServerAddress    string `json:"server_address,omitempty"     mapstructure:"server_address"`
	TlsSkipVerify    bool   `json:"tls_skip_verify,omitempty"    mapstructure:"tls_skip_verify"`
	TlsCertFile      string `json:"tls_cert_file,omitempty"      mapstructure:"tls_cert_file"`
	TlsKeyFile       string `json:"tls_key_file,omitempty"       mapstructure:"tls_key_file"`
	TlsCAFile        string `json:"tls_ca_file,omitempty"        mapstructure:"tls_ca_file"`
	SpiffeSocketPath string `json:"spiffe_socket_path,omitempty" mapstructure:"spiffe_socket_path"`
	SpiffeToken      string `json:"spiffe_token,omitempty"       mapstructure:"spiffe_token"`
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

	_ = v.BindEnv("tls_skip_verify")
	v.SetDefault("tls_skip_verify", DefaultTlsSkipVerify)

	_ = v.BindEnv("spiffe_socket_path")
	v.SetDefault("spiffe_socket_path", "")

	_ = v.BindEnv("spiffe_token")
	v.SetDefault("spiffe_token", "")

	_ = v.BindEnv("auth_mode")
	v.SetDefault("auth_mode", "")

	_ = v.BindEnv("jwt_audience")
	v.SetDefault("jwt_audience", "")

	_ = v.BindEnv("tls_cert_file")
	v.SetDefault("tls_cert_file", "")

	_ = v.BindEnv("tls_key_file")
	v.SetDefault("tls_key_file", "")

	_ = v.BindEnv("tls_ca_file")
	v.SetDefault("tls_ca_file", "")

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
