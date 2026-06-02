// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

// Client holds the merged CLI config (from config file, env, and flags).
// It is populated by cmd/options.go init and updated when flags are parsed.
var Client *client.Config = &client.DefaultConfig

// Context is the per-command context override.
var Context string

// ChangedClientConfigFields returns schema field names for explicitly set client flags.
func ChangedClientConfigFields(cmd *cobra.Command) []string {
	//nolint:gosec // G101: These are configuration field names, not credential values.
	flagToField := map[string]string{
		"server-addr":        "server_address",
		"auth-mode":          "auth_mode",
		"spiffe-socket-path": "spiffe_socket_path",
		"jwt-audience":       "jwt_audience",
		"tls-skip-verify":    "tls_skip_verify",
		"tls-ca-file":        "tls_ca_file",
		"tls-cert-file":      "tls_cert_file",
		"tls-key-file":       "tls_key_file",
		"oidc-issuer":        "oidc_issuer",
		"oidc-client-id":     "oidc_client_id",
		"auth-token":         "auth_token",
	}

	fields := make([]string, 0, len(flagToField))

	flags := cmd.Root().PersistentFlags()
	for flagName, fieldName := range flagToField {
		flag := flags.Lookup(flagName)
		if flag != nil && flag.Changed {
			fields = append(fields, fieldName)
		}
	}

	return fields
}
