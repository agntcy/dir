// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logout

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/agntcy/dir/cli/hub/idp"
	secretstore2 "github.com/agntcy/dir/cli/hub/sessionstore"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var ErrSecretNotFoundForAddress = errors.New("no active session found for the address. please login first")

func NewCommand(opts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Agent Hub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get current hub secret from context
			secret, ok := ctxUtils.GetCurrentHubSecretFromContext(cmd.Context())
			if !ok {
				return ErrSecretNotFoundForAddress
			}

			// Get secret store from context
			secretStore, ok := ctxUtils.GetSessionStoreFromContext(cmd.Context())
			if !ok {
				return errors.New("failed to get secret store from context")
			}

			idpClient, ok := ctxUtils.GetIdpClientFromContext(cmd.Context())
			if !ok {
				return errors.New("failed to get idp client from context")
			}

			return runCmd(cmd.OutOrStdout(), opts, secret, secretStore, idpClient)
		},
		TraverseChildren: true,
	}

	return cmd
}

func runCmd(outStream io.Writer, opts *options.HubOptions, secret *secretstore2.HubSession, secretStore secretstore2.SessionStore, idpClient idp.Client) error {
	resp, err := idpClient.Logout(&idp.LogoutRequest{IDToken: secret.IDToken})
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	if resp.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to logout: unexpected status code: %d: %s", resp.Response.StatusCode, resp.Body)
	}

	// Remove secret from secret store
	if err = secretStore.RemoveHubSession(opts.ServerAddress); err != nil {
		return fmt.Errorf("failed to remove secret from secret store: %w", err)
	}

	fmt.Fprintln(outStream, "Successfully logged out.")

	return nil
}
