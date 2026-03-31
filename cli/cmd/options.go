// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
)

func init() {
	// load config
	if cfg, err := client.LoadConfig(); err == nil {
		config.Client = cfg
	}

	// set flags
	flags := RootCmd.PersistentFlags()
	flags.StringVar(&config.Client.ServerAddress, "server-addr", config.Client.ServerAddress, "Directory Server API address")
	flags.StringVar(&config.Client.AuthMode, "auth-mode", config.Client.AuthMode, "Authentication mode: x509, jwt, token (SPIFFE), tls, oidc, insecure, none, or empty for auto-detect")
	flags.StringVar(&config.Client.SpiffeSocketPath, "spiffe-socket-path", config.Client.SpiffeSocketPath, "Path to SPIFFE Workload API socket (for x509 or JWT authentication)")
	flags.StringVar(&config.Client.SpiffeToken, "spiffe-token", config.Client.SpiffeToken, "Path to JSON file containing SPIFFE X509 SVID token (for --auth-mode=token)")
	flags.StringVar(&config.Client.JWTAudience, "jwt-audience", config.Client.JWTAudience, "JWT audience (for JWT authentication mode)")
	flags.BoolVar(&config.Client.TlsSkipVerify, "tls-skip-verify", config.Client.TlsSkipVerify, "Skip TLS verification (for TLS authentication mode)")
	flags.StringVar(&config.Client.TlsCAFile, "tls-ca-file", config.Client.TlsCAFile, "Path to TLS CA file (for TLS authentication mode)")
	flags.StringVar(&config.Client.TlsCertFile, "tls-cert-file", config.Client.TlsCertFile, "Path to TLS certificate file (for TLS authentication mode)")
	flags.StringVar(&config.Client.TlsKeyFile, "tls-key-file", config.Client.TlsKeyFile, "Path to TLS key file (for TLS authentication mode)")
	flags.StringVar(&config.Client.OIDCIssuer, "oidc-issuer", config.Client.OIDCIssuer, "OIDC issuer URL (e.g. https://dex.example.com) for interactive login")
	flags.StringVar(&config.Client.OIDCClientID, "oidc-client-id", config.Client.OIDCClientID, "OIDC client ID for interactive login (PKCE)")
	flags.StringVar(&config.Client.AuthToken, "auth-token", config.Client.AuthToken, "Pre-issued Bearer token (JWT). Useful for CI and non-interactive workflows; no login needed")

	// mark required flags
	RootCmd.MarkFlagRequired("server-addr") //nolint:errcheck
}
