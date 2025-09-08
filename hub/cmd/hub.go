// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package cmd provides the CLI commands for the Agent Hub, including login, logout, push, pull, and org management.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/apikey"
	"github.com/agntcy/dir/hub/cmd/info"
	"github.com/agntcy/dir/hub/cmd/login"
	"github.com/agntcy/dir/hub/cmd/logout"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/cmd/orgs"
	"github.com/agntcy/dir/hub/cmd/pull"
	"github.com/agntcy/dir/hub/cmd/push"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/agntcy/dir/hub/utils/token"
	"github.com/spf13/cobra"
)

// NewHubCommand creates the root "hub" command for the Agent Hub CLI.
// It sets up persistent pre-run logic for session/config loading and token refresh,
// attaches the session to the command context, and adds all subcommands (login, logout, push, pull, orgs).
// Returns the configured *cobra.Command.
func NewHubCommand(ctx context.Context, baseOption *options.BaseOption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Agent Hub",

		TraverseChildren: true,
	}

	cmd.SetContext(ctx)

	opts := options.NewHubOptions(baseOption, cmd)

	var noCache bool

	cmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Skip session file operations and use only API key authentication")

	//nolint:contextcheck // context is set via cmd.SetContext(ctx) and accessed via cmd.Context()
	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		cmd.SetOut(os.Stdout)
		cmd.SetErr(os.Stderr)
		opts.Complete()

		sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())

		currentSession, err := sessionStore.GetHubSession(opts.ServerAddress)
		if err != nil && !errors.Is(err, sessionstore.ErrSessionNotFound) {
			return fmt.Errorf("failed to get hub session: %w", err)
		}

		if currentSession == nil {
			currentSession = &sessionstore.HubSession{}
		}

		authConfig, err := config.FetchAuthConfig(cmd.Context(), opts.ServerAddress)
		if err != nil {
			return fmt.Errorf("failed to fetch auth config: %w", err)
		}

		currentSession.AuthConfig = &sessionstore.AuthConfig{
			ClientID:           authConfig.ClientID,
			IdpProductID:       authConfig.IdpProductID,
			IdpFrontendAddress: authConfig.IdpFrontendAddress,
			IdpBackendAddress:  authConfig.IdpBackendAddress,
			IdpIssuerAddress:   authConfig.IdpIssuerAddress,
			HubBackendAddress:  authConfig.HubBackendAddress,
			APIKeyClientID:     authConfig.APIKeyClientID,
		}

		// Only refresh token if not running login or logout
		if cmd.Name() != "login" && cmd.Name() != "logout" {
			oktaClient := okta.NewClient(authConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())
			if err := token.RefreshTokenIfExpired(opts.ServerAddress, currentSession, sessionStore, oktaClient); err != nil {
				return fmt.Errorf("failed to refresh expired access token: %w", err)
			}
		}

		// If --no-cache is specified, skip all session file operations
		if cmd.Flags().Changed("no-cache") {
			fmt.Fprintf(cmd.OutOrStdout(), "Skipping session file operations due to --no-cache flag\n")
		} else {
			if err := sessionStore.SaveHubSession(opts.ServerAddress, currentSession); err != nil {
				return fmt.Errorf("failed to save updated session with auth config: %w", err)
			}
		}

		// Attach the session to cmd.Context()
		ctx := context.WithValue(cmd.Context(), sessionstore.SessionContextKey, currentSession)
		cmd.SetContext(ctx)

		return nil
	}

	cmd.AddCommand(
		login.NewCommand(opts),
		logout.NewCommand(opts),
		push.NewCommand(opts),
		pull.NewCommand(opts),
		orgs.NewCommand(opts),
		apikey.NewCommand(opts),
		info.NewCommand(opts),
	)

	return cmd
}
