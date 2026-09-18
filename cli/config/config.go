// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"

	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

// Client holds the merged CLI config (from config file, env, and flags).
// It is populated by cmd/options.go init and updated when flags are parsed.
var Client *client.Config = &client.DefaultConfig

// Context is the per-command context override.
var Context string

// ResolveClient merges this invocation's client flags over the context it
// names — `--context`, or current_context when the flag is unset — and returns
// the result.
//
// It is a pure function: the package-level Client is left alone, so a caller
// that only wants to read the resolved address does not change what the rest of
// the run sees. Root assigns the result to Client itself, because the flag
// bindings point at that value.
//
// Anything deriving a Directory address must come through here rather than
// calling clientconfig.Resolve with an empty Context, or it silently reports
// current_context while the invocation is talking to another Directory
// entirely.
func ResolveClient(cmd *cobra.Command) (*client.Config, error) {
	return resolveClient(cmd, false)
}

// ResolveClientLenient is ResolveClient with validation skipped, for a caller
// that only needs to read the effective connection settings and would rather
// have a partially-set or forward-compatible config than none at all —
// projecting the address into an installed MCP entry, say, where falling back
// to a default would silently point it somewhere else.
//
// Never hand its result to client.New: nothing has checked that the settings
// hang together.
func ResolveClientLenient(cmd *cobra.Command) (*client.Config, error) {
	return resolveClient(cmd, true)
}

func resolveClient(cmd *cobra.Command, skipValidation bool) (*client.Config, error) {
	fields := ChangedClientConfigFields(cmd)

	var overrides *client.Config
	if len(fields) > 0 {
		overrides = Client
	}

	cfg, _, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		Context:            Context,
		Overrides:          overrides,
		OverrideFields:     fields,
		SkipValidation:     skipValidation,
		AllowUnknownFields: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve client config: %w", err)
	}

	return cfg, nil
}

// ChangedClientConfigFields returns schema field names for explicitly set client flags.
func ChangedClientConfigFields(cmd *cobra.Command) []string {
	//nolint:gosec // G101: These are configuration field names, not credential values.
	flagToField := map[string]string{
		"server-addr":        "server_address",
		"auth-mode":          "auth_mode",
		"spiffe-socket-path": "spiffe_socket_path",
		"spiffe-token":       "spiffe_token",
		"jwt-audience":       "jwt_audience",
		"tls-skip-verify":    "tls_skip_verify",
		"tls-ca-file":        "tls_ca_file",
		"tls-cert-file":      "tls_cert_file",
		"tls-key-file":       "tls_key_file",
		"oidc-issuer":        "oidc_issuer",
		"oidc-client-id":     "oidc_client_id",
		"oidc-scopes":        "oidc_scopes",
		"oidc-audience":      "oidc_audience",
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
