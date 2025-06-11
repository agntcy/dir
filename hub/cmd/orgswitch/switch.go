// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package orgswitch provides the CLI subcommand for switching between organizations (tenants) for the logged-in user.
// This command is intended to be used as "orgs switch" to enable switching, not listing.
package orgswitch

import (
	"errors"
	"fmt"

	auth "github.com/agntcy/dir/hub/auth"
	idp "github.com/agntcy/dir/hub/client/idp"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	fileUtils "github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/spf13/cobra"
)

// NewCommand creates the "switch" subcommand for switching between organizations (tenants).
// It enables the user to select or specify an organization to switch to, updates the session, and saves it.
// This command is intended to be used as "orgs switch" and does not list organizations.
// Returns the configured *cobra.Command.
func NewCommand(hubOpts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch [flags]",
		Short: "Switch between organizations of logged-in user",
		Long: `
This command help switching between logged-in user's orgs. You need to log in first with
'dirctl hub login' command. If --org flag is specified user will be logged in to the specified
organization. In any other case, org could be selected from an interactive list.
`,
	}

	opts := options.NewTenantSwitchOptions(hubOpts, cmd)

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		// Retrieve session from context
		ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
		currentSession, ok := ctxSession.(*sessionstore.HubSession)

		if !ok || currentSession == nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Could not get current session\n")

			return errors.New("could not get current session")
		}
		// Load session store for saving
		sessionStore := sessionstore.NewFileSessionStore(fileUtils.GetSessionFilePath())

		// Load tenants directly using idp client
		idpClient := idp.NewClient(currentSession.AuthConfig.IdpBackendAddress, httpUtils.CreateSecureHTTPClient())
		accessToken := currentSession.Tokens[currentSession.CurrentTenant].AccessToken
		productID := currentSession.AuthConfig.IdpProductID

		idpResp, err := idpClient.GetTenantsInProduct(cmd.Context(), productID, idp.WithBearerToken(accessToken))
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Could not fetch tenants: %v\n", err)

			return fmt.Errorf("could not fetch tenants: %w", err)
		}

		if idpResp.TenantList == nil {
			fmt.Fprintf(cmd.OutOrStderr(), "No tenants found for this user.\n")

			return errors.New("no tenants found")
		}

		tenants := idpResp.TenantList.Tenants

		oktaClient := okta.NewClient(currentSession.AuthConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())

		updatedSession, msg, err := auth.SwitchTenant(cmd.Context(), opts, tenants, currentSession, oktaClient)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "An error occurred during org switch. Try to call `dirctl hub login` to solve the issue.\nError details: %v\n", err)

			return fmt.Errorf("failed to switch tenant: %w", err)
		}

		if err := sessionStore.SaveHubSession(opts.ServerAddress, updatedSession); err != nil {
			return fmt.Errorf("could not save session to session store: %w", err)
		}

		if msg != "" {
			fmt.Fprintln(cmd.OutOrStdout(), msg)
		}

		return nil
	}

	return cmd
}
