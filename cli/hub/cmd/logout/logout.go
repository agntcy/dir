// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logout

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/agntcy/dir/cli/hub/client/okta"
	"github.com/agntcy/dir/cli/hub/cmd/options"
	"github.com/agntcy/dir/cli/hub/sessionstore"
	ctxUtils "github.com/agntcy/dir/cli/hub/utils/context"
	"github.com/spf13/cobra"
)

var ErrSecretNotFoundForAddress = errors.New("no active session found for the address. please login first")

func NewCommand(opts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Agent Hub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get current hub session from context
			session, ok := ctxUtils.GetCurrentHubSessionFromContext(cmd)
			if !ok {
				return ErrSecretNotFoundForAddress
			}

			// Get session store from context
			sessionStore, ok := ctxUtils.GetSessionStoreFromContext(cmd)
			if !ok {
				return errors.New("failed to get session store from context")
			}

			oktaClient, ok := ctxUtils.GetOktaClientFromContext(cmd)
			if !ok {
				return errors.New("failed to get okta client from context")
			}

			return runCmd(cmd.OutOrStdout(), opts, session, sessionStore, oktaClient)
		},
		TraverseChildren: true,
	}

	return cmd
}

func runCmd(outStream io.Writer, opts *options.HubOptions, session *sessionstore.HubSession, secretStore sessionstore.SessionStore, oktaClient okta.Client) error {
	resp, err := oktaClient.Logout(&okta.LogoutRequest{IDToken: session.Tokens[session.CurrentTenant].IDToken})
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to logout: unexpected status code: %d: %s", resp.Response.StatusCode, resp.Body)
	}

	// Remove session from session store
	if err = secretStore.RemoveSession(opts.ServerAddress); err != nil {
		return fmt.Errorf("failed to remove session from session store: %w", err)
	}

	fmt.Fprintln(outStream, "Successfully logged out.")

	return nil
}
