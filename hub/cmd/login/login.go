// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package login provides the CLI command for logging in to the Agent Hub.
package login

import (
	"errors"
	"fmt"

	auth "github.com/agntcy/dir/hub/auth"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	"github.com/agntcy/dir/hub/utils/file"
	"github.com/agntcy/dir/hub/utils/http"
	"github.com/spf13/cobra"
)

// NewCommand creates the "login" command for the Agent Hub CLI.
// It handles user authentication, session creation, and token storage.
// Returns the configured *cobra.Command.
func NewCommand(hubOptions *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "login",
		Short:            "Login to the Agent Hub",
		TraverseChildren: true,
	}

	opts := options.NewLoginOptions(hubOptions)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || currentSession == nil {
			return errors.New("failed to get current session from context")
		}
		// Load session store for saving
		sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())
		// Construct Okta client
		oktaClient := okta.NewClient(currentSession.AuthConfig.IdpIssuerAddress, http.CreateSecureHTTPClient())
		// Call auth.Login with loaded objects
		updatedSession, err := auth.Login(cmd.Context(), oktaClient, currentSession)
		if err != nil {
			return fmt.Errorf("failed to login: %w", err)
		}

		if err := sessionStore.SaveHubSession(opts.ServerAddress, updatedSession); err != nil {
			return fmt.Errorf("failed to save tokens: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Successfully logged in to Agent Hub\nAddress: %s\nUser: %s\nOrganization: %s\n", opts.ServerAddress, updatedSession.User, updatedSession.CurrentTenant)

		return nil
	}

	return cmd
}
