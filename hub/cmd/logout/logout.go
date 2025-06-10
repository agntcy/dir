// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package logout

import (
	"errors"

	auth "github.com/agntcy/dir/hub/auth"
	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/sessionstore"
	fileUtils "github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/spf13/cobra"
)

var ErrSecretNotFoundForAddress = errors.New("no active session found for the address. please login first")

func NewCommand(opts *options.HubOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Logout from Agent Hub",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Retrieve session from context
			ctxSession := cmd.Context().Value(sessionstore.SessionContextKey)
			currentSession, ok := ctxSession.(*sessionstore.HubSession)
			if !ok || currentSession == nil {
				return ErrSecretNotFoundForAddress
			}
			// Load session store for removal
			sessionStore := sessionstore.NewFileSessionStore(fileUtils.GetSessionFilePath())
			oktaClient := okta.NewClient(currentSession.AuthConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())
			return auth.Logout(cmd.OutOrStdout(), opts, currentSession, sessionStore, oktaClient)
		},
		TraverseChildren: true,
	}

	return cmd
}
