// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package hub

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/cmd/hub/login"
	"github.com/agntcy/dir/cli/cmd/hub/logout"
	"github.com/agntcy/dir/cli/cmd/hub/pull"
	"github.com/agntcy/dir/cli/cmd/hub/push"
	"github.com/agntcy/dir/cli/cmd/hub/tenants"
	"github.com/agntcy/dir/cli/hub/config"
	"github.com/agntcy/dir/cli/hub/okta"
	"github.com/agntcy/dir/cli/hub/sessionstore"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
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
		sessionStore, ok := ctxUtils.GetSessionStoreFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get session store from context")
		}

		session, err := sessionStore.GetHubSession(opts.ServerAddress)
		if err != nil && !errors.Is(err, sessionstore.ErrSessionNotFound) {
			return fmt.Errorf("failed to get hub session: %w", err)
		}

		if session == nil {
			session = &sessionstore.HubSession{}
		}

		authConfig, err := config.FetchAuthConfig(opts.ServerAddress)
		if err != nil {
			return fmt.Errorf("failed to fetch auth config: %w", err)
		}

		session.AuthConfig = &sessionstore.AuthConfig{
			ClientID:           authConfig.ClientID,
			IdpProductID:       authConfig.IdpProductID,
			IdpFrontendAddress: authConfig.IdpFrontendAddress,
			IdpBackendAddress:  authConfig.IdpBackendAddress,
			IdpIssuerAddress:   authConfig.IdpIssuerAddress,
			HubBackendAddress:  authConfig.HubBackendAddress,
		}

		ctx := cmd.Context()
		ctx = ctxUtils.SetCurrentHubSessionForContext(ctx, session)

		idpClient := okta.NewClient(session.IdpIssuerAddress)
		ctx = ctxUtils.SetOktaClientForContext(ctx, idpClient)

		cmd.SetContext(ctx)

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
