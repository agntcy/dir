// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"github.com/agntcy/dir/client"
)

var clientConfig = &client.DefaultConfig

func init() {
	// load config
	if cfg, err := client.LoadConfig(); err == nil {
		clientConfig = cfg
	}

	// set flags
	flags := RootCmd.PersistentFlags()
	flags.StringVar(&clientConfig.ServerAddress, "server-addr", clientConfig.ServerAddress, "Directory Server API address")
	flags.StringVar(&clientConfig.AuthMode, "auth-mode", clientConfig.AuthMode, "Authentication mode: none, x509, jwt, token, tls")
	flags.StringVar(&clientConfig.SpiffeSocketPath, "spiffe-socket-path", clientConfig.SpiffeSocketPath, "Path to SPIFFE Workload API socket (for x509 or JWT authentication)")
	flags.StringVar(&clientConfig.SpiffeToken, "spiffe-token", clientConfig.SpiffeToken, "Path to file containing SPIFFE X509 SVID token (for token authentication)")
	flags.StringVar(&clientConfig.JWTAudience, "jwt-audience", clientConfig.JWTAudience, "JWT audience (for JWT authentication mode)")
	flags.BoolVar(&clientConfig.TlsSkipVerify, "tls-skip-verify", clientConfig.TlsSkipVerify, "Skip TLS verification (for TLS authentication mode)")
	flags.StringVar(&clientConfig.TlsCAFile, "tls-ca-file", clientConfig.TlsCAFile, "Path to TLS CA file (for TLS authentication mode)")
	flags.StringVar(&clientConfig.TlsCertFile, "tls-cert-file", clientConfig.TlsCertFile, "Path to TLS certificate file (for TLS authentication mode)")
	flags.StringVar(&clientConfig.TlsKeyFile, "tls-key-file", clientConfig.TlsKeyFile, "Path to TLS key file (for TLS authentication mode)")

	// mark required flags
	RootCmd.MarkFlagRequired("server-addr") //nolint:errcheck
}
