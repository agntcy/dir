// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package logout provides the CLI command for logging out of the Agent Hub.
package logout

import (
	"errors"
	"fmt"

	auth "github.com/agntcy/dir/hub/auth"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	fileUtils "github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/spf13/cobra"
)

var ErrSecretNotFoundForAddress = errors.New("no active session found for the address. please login first")

// NewCommand creates the "logout" command for the Agent Hub CLI.
// It handles user logout, session removal, and token revocation.
// Returns the configured *cobra.Command.
func NewCommand(opts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Agent Hub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Retrieve session from context
			ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
			currentSession, ok := ctxSession.(*sessionstore.HubSession)

			if !ok || !auth.HasLoginCreds(currentSession) {
				return ErrSecretNotFoundForAddress
			}
			// Load session store for removal
			sessionStore := sessionstore.NewFileSessionStore(fileUtils.GetSessionFilePath())
			oktaClient := okta.NewClient(currentSession.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())

			err := auth.Logout(opts, currentSession, sessionStore, oktaClient)
			if err != nil {
				return fmt.Errorf("failed to logout: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "Successfully logged out from Agent Hub")

			return nil
		},
		TraverseChildren: true,
	}

	return cmd
}
