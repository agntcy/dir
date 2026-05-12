// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

const redacted = "<redacted>"

var showCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Show a context with sensitive values redacted",
	Long: `Show an effective dirctl client context.

This command resolves a context using the shared client config resolver and
prints the effective client settings. Sensitive values such as bearer tokens
and SPIFFE token paths are redacted from output.

If [name] is provided, that context is shown. If omitted, the command uses the
same context selection rules as other dirctl commands. Environment variables
and explicitly changed root flags are included in the effective output.

Arguments:

[name] is optional. When omitted, the active context is selected from --context,
DIRECTORY_CLIENT_CONTEXT, or current_context.

Examples:

1. Show the active context:
   dirctl context show

2. Show a specific context:
   dirctl context show prod

3. Preview a context with a one-off server override:
   dirctl --server-addr localhost:8888 context show dev`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShow,
}

func runShow(cmd *cobra.Command, args []string) error {
	contextName := selectedShowContext(args)
	fields := config.ChangedClientConfigFields(cmd)

	var overrides *client.Config
	if len(fields) > 0 {
		overrides = config.Client
	}

	cfg, resolved, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		Context:        contextName,
		Overrides:      overrides,
		OverrideFields: fields,
	})
	if err != nil {
		return fmt.Errorf("failed to show context: %w", err)
	}

	if resolved.Name == "" {
		return errors.New("no context selected")
	}

	printResolvedConfig(cmd, resolved, cfg)

	return nil
}

func selectedShowContext(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return config.Context
}

func printResolvedConfig(cmd *cobra.Command, resolved *clientconfig.ResolvedContext, cfg *client.Config) {
	cmd.Printf("name: %s\n", resolved.Name)
	cmd.Printf("source: %s\n", resolved.Source)
	cmd.Printf("path: %s\n", resolved.Path)
	cmd.Println("config:")

	values := resolvedConfigValues(cfg)
	keys := sortedValueKeys(values)

	for _, key := range keys {
		if values[key] == "" {
			continue
		}

		cmd.Printf("  %s: %s\n", key, values[key])
	}
}

func resolvedConfigValues(cfg *client.Config) map[string]string {
	return map[string]string{
		"auth_mode":          cfg.AuthMode,
		"auth_token":         redact(cfg.AuthToken),
		"jwt_audience":       cfg.JWTAudience,
		"oidc_client_id":     cfg.OIDCClientID,
		"oidc_issuer":        cfg.OIDCIssuer,
		"server_address":     cfg.ServerAddress,
		"spiffe_socket_path": cfg.SpiffeSocketPath,
		"spiffe_token":       redact(cfg.SpiffeToken),
		"tls_ca_file":        cfg.TlsCAFile,
		"tls_cert_file":      cfg.TlsCertFile,
		"tls_key_file":       cfg.TlsKeyFile,
		"tls_skip_verify":    fmt.Sprintf("%t", cfg.TlsSkipVerify),
	}
}

func sortedValueKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	return keys
}

func redact(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}

	return redacted
}
