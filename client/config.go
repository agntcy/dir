// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix = "DIRECTORY_CLIENT"

	DefaultServerAddress = "0.0.0.0:8888"
	DefaultTlsSkipVerify = false
	DefaultCallbackPort  = 8484
	// DefaultOIDCRedirectURI is the default OAuth callback URL (not a credential).
	//nolint:gosec // G101: redirect URI is not a secret; it's registered with the IdP
	DefaultOIDCRedirectURI = "http://localhost:8484/callback"
	DefaultOAuthTimeout    = 5 * time.Minute
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

	// OIDC configuration (for interactive login and CI token)
	OIDCIssuer      string `json:"oidc_issuer,omitempty"       mapstructure:"oidc_issuer"`
	OIDCClientID    string `json:"oidc_client_id,omitempty"    mapstructure:"oidc_client_id"`
	OIDCToken       string `json:"oidc_token,omitempty"        mapstructure:"oidc_token"`
	OIDCRedirectURI string `json:"oidc_redirect_uri,omitempty" mapstructure:"oidc_redirect_uri"`

	// OIDC machine/service-user configuration (client credentials flow).
	OIDCMachineClientID         string   `json:"oidc_machine_client_id,omitempty"          mapstructure:"oidc_machine_client_id"`
	OIDCMachineClientSecret     string   `json:"oidc_machine_client_secret,omitempty"      mapstructure:"oidc_machine_client_secret"`
	OIDCMachineClientSecretFile string   `json:"oidc_machine_client_secret_file,omitempty" mapstructure:"oidc_machine_client_secret_file"`
	OIDCMachineScopes           []string `json:"oidc_machine_scopes,omitempty"             mapstructure:"oidc_machine_scopes"`
	OIDCMachineTokenEndpoint    string   `json:"oidc_machine_token_endpoint,omitempty"     mapstructure:"oidc_machine_token_endpoint"`
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

	_ = v.BindEnv("oidc_issuer")
	v.SetDefault("oidc_issuer", "")

	_ = v.BindEnv("oidc_client_id")
	v.SetDefault("oidc_client_id", "")

	_ = v.BindEnv("oidc_token")
	v.SetDefault("oidc_token", "")

	_ = v.BindEnv("oidc_redirect_uri")
	v.SetDefault("oidc_redirect_uri", DefaultOIDCRedirectURI)

	_ = v.BindEnv("oidc_machine_client_id")
	v.SetDefault("oidc_machine_client_id", "")

	_ = v.BindEnv("oidc_machine_client_secret")
	v.SetDefault("oidc_machine_client_secret", "")

	_ = v.BindEnv("oidc_machine_client_secret_file")
	v.SetDefault("oidc_machine_client_secret_file", "")

	_ = v.BindEnv("oidc_machine_scopes")
	v.SetDefault("oidc_machine_scopes", []string{})

	_ = v.BindEnv("oidc_machine_token_endpoint")
	v.SetDefault("oidc_machine_token_endpoint", "")

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
