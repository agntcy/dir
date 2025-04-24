// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/hub/client/okta"
	"github.com/agntcy/dir/hub/cmd/login"
	"github.com/agntcy/dir/hub/cmd/logout"
	"github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/cmd/pull"
	"github.com/agntcy/dir/hub/cmd/push"
	"github.com/agntcy/dir/hub/cmd/tenants"
	"github.com/agntcy/dir/hub/config"
	"github.com/agntcy/dir/hub/sessionstore"
	ctxUtils "github.com/agntcy/dir/hub/utils/context"
	"github.com/agntcy/dir/hub/utils/file"
	httpUtils "github.com/agntcy/dir/hub/utils/http"
	"github.com/spf13/cobra"
)

func NewHubCommand(baseOption *options.BaseOption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Agent Hub",

		TraverseChildren: true,
	}

	opts := options.NewHubOptions(baseOption, cmd)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		opts.Complete()

		sessionStore := sessionstore.NewFileSessionStore(file.GetSessionFilePath())

		currentSession, err := sessionStore.GetHubSession(opts.ServerAddress)
		if err != nil && !errors.Is(err, sessionstore.ErrSessionNotFound) {
			return fmt.Errorf("failed to get hub session: %w", err)
		}

		if currentSession == nil {
			currentSession = &sessionstore.HubSession{}
		}

		authConfig, err := config.FetchAuthConfig(opts.ServerAddress)
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
		}

		if ok := ctxUtils.SetSessionStoreForContext(cmd, sessionStore); !ok {
			return errors.New("failed to set session store for context")
		}

		if ok := ctxUtils.SetCurrentHubSessionForContext(cmd, currentSession); !ok {
			return errors.New("failed to set current hub session for context")
		}

		oktaClient := okta.NewClient(authConfig.IdpIssuerAddress, httpUtils.CreateSecureHTTPClient())
		if ok := ctxUtils.SetOktaClientForContext(cmd, oktaClient); !ok {
			return errors.New("failed to set okta client for context")
		}

		return nil
	}

	cmd.AddCommand(
		login.NewCommand(opts),
		logout.NewCommand(opts),
		push.NewCommand(opts),
		pull.NewCommand(opts),
		tenants.NewCommand(opts),
	)

	return cmd
}
