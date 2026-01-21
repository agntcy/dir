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
	flags.StringVar(&clientConfig.AuthMode, "auth-mode", clientConfig.AuthMode, "Authentication mode: x509, jwt, token (SPIFFE), tls, github, insecure, none, or empty for auto-detect")
	flags.StringVar(&clientConfig.GitHubToken, "github-token", clientConfig.GitHubToken, "GitHub token (PAT or OAuth) for authentication - useful for CI/CD (can also use DIRECTORY_CLIENT_GITHUB_TOKEN env var)")
	flags.StringVar(&clientConfig.SpiffeSocketPath, "spiffe-socket-path", clientConfig.SpiffeSocketPath, "Path to SPIFFE Workload API socket (for x509 or JWT authentication)")
	flags.StringVar(&clientConfig.SpiffeToken, "spiffe-token", clientConfig.SpiffeToken, "Path to JSON file containing SPIFFE X509 SVID token (for --auth-mode=token)")
	flags.StringVar(&clientConfig.JWTAudience, "jwt-audience", clientConfig.JWTAudience, "JWT audience (for JWT authentication mode)")
	flags.BoolVar(&clientConfig.TlsSkipVerify, "tls-skip-verify", clientConfig.TlsSkipVerify, "Skip TLS verification (for TLS authentication mode)")
	flags.StringVar(&clientConfig.TlsCAFile, "tls-ca-file", clientConfig.TlsCAFile, "Path to TLS CA file (for TLS authentication mode)")
	flags.StringVar(&clientConfig.TlsCertFile, "tls-cert-file", clientConfig.TlsCertFile, "Path to TLS certificate file (for TLS authentication mode)")
	flags.StringVar(&clientConfig.TlsKeyFile, "tls-key-file", clientConfig.TlsKeyFile, "Path to TLS key file (for TLS authentication mode)")
	flags.StringVar(&clientConfig.GitHubClientID, "github-client-id", clientConfig.GitHubClientID, "GitHub OAuth App client ID (for github authentication mode)")
	flags.StringVar(&clientConfig.GitHubClientSecret, "github-client-secret", clientConfig.GitHubClientSecret, "GitHub OAuth App client secret (for github authentication mode)")

	// mark required flags
	RootCmd.MarkFlagRequired("server-addr") //nolint:errcheck
}
