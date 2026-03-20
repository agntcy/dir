// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/agntcy/dir/cli/config"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

const machineAuthTimeout = 15 * time.Second

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Authenticate non-interactively with OIDC client credentials",
	Long: `Authenticate a machine/service-user identity with OIDC client credentials.

This command is intended for service users and automation, unlike 'dirctl auth login'
which is an interactive PKCE flow for human users.

Required configuration (flags/env):
  --oidc-issuer, DIRECTORY_CLIENT_OIDC_ISSUER
  --oidc-machine-client-id, DIRECTORY_CLIENT_OIDC_MACHINE_CLIENT_ID
  One of:
    --oidc-machine-client-secret, DIRECTORY_CLIENT_OIDC_MACHINE_CLIENT_SECRET
    --oidc-machine-client-secret-file, DIRECTORY_CLIENT_OIDC_MACHINE_CLIENT_SECRET_FILE

Examples:
  # Authenticate with client secret from env
  DIRECTORY_CLIENT_OIDC_ISSUER=https://dev.idp.ads.outshift.io \
  DIRECTORY_CLIENT_OIDC_MACHINE_CLIENT_ID=<client-id> \
  DIRECTORY_CLIENT_OIDC_MACHINE_CLIENT_SECRET=<secret> \
  dirctl auth machine

  # Authenticate with client secret from file
  dirctl auth machine \
    --oidc-issuer https://dev.idp.ads.outshift.io \
    --oidc-machine-client-id <client-id> \
    --oidc-machine-client-secret-file /path/to/secret`,
	RunE: runMachineAuth,
}

func runMachineAuth(cmd *cobra.Command, _ []string) error {
	cfg := config.Client

	if cfg.OIDCIssuer == "" {
		return errors.New("OIDC issuer is required for machine authentication (set --oidc-issuer or DIRECTORY_CLIENT_OIDC_ISSUER)")
	}

	if cfg.OIDCMachineClientID == "" {
		return errors.New("OIDC machine client ID is required (set --oidc-machine-client-id or DIRECTORY_CLIENT_OIDC_MACHINE_CLIENT_ID)")
	}

	hasSecret := cfg.OIDCMachineClientSecret != ""

	hasSecretFile := cfg.OIDCMachineClientSecretFile != ""
	if !hasSecret && !hasSecretFile {
		return errors.New("OIDC machine client secret is required (set --oidc-machine-client-secret, --oidc-machine-client-secret-file, or corresponding env vars)")
	}

	token, err := client.OIDC.RunClientCredentialsFlow(cmd.Context(), &client.ClientCredentialsConfig{
		Issuer:           cfg.OIDCIssuer,
		TokenEndpoint:    cfg.OIDCMachineTokenEndpoint,
		ClientID:         cfg.OIDCMachineClientID,
		ClientSecret:     cfg.OIDCMachineClientSecret,
		ClientSecretFile: cfg.OIDCMachineClientSecretFile,
		Scopes:           cfg.OIDCMachineScopes,
		Timeout:          machineAuthTimeout,
	})
	if err != nil {
		return fmt.Errorf("failed to authenticate machine identity: %w", err)
	}

	cache := client.NewTokenCache()

	userID := token.Subject
	if userID == "" {
		userID = cfg.OIDCMachineClientID
	}

	cached := &client.CachedToken{
		AccessToken:  token.AccessToken,
		TokenType:    token.TokenType,
		Provider:     "oidc",
		Issuer:       cfg.OIDCIssuer,
		RefreshToken: "",
		ExpiresAt:    token.ExpiresAt,
		User:         cfg.OIDCMachineClientID,
		UserID:       userID,
		Email:        "",
	}

	if err := cache.Save(cached); err != nil {
		return fmt.Errorf("failed to cache machine token: %w", err)
	}

	cmd.Println("Machine authentication successful; token cached.")

	return nil
}
