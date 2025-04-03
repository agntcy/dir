package hub

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/cmd/hub/login"
	"github.com/agntcy/dir/cli/cmd/hub/logout"
	"github.com/agntcy/dir/cli/cmd/hub/pull"
	"github.com/agntcy/dir/cli/cmd/hub/push"
	"github.com/agntcy/dir/cli/hub/config"
	"github.com/agntcy/dir/cli/hub/idp"
	secretstore2 "github.com/agntcy/dir/cli/hub/secretstore"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

func NewHubCommand(baseOption *options.BaseOption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Agent Hub",

		TraverseChildren: true,
	}

	opts := options.NewHubOptions(baseOption, cmd)

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
		if !ok {
			return fmt.Errorf("failed to get secret store from context")
		}

		ctx := cmd.Context()
		var secret *secretstore2.HubSecret
		var err error

		secret, err = secretStore.GetHubSecret(opts.ServerAddress)
		if err != nil && !errors.Is(err, secretstore2.ErrSecretNotFound) {
			return err
		}

		if secret == nil {
			var authConfig *config.AuthConfig
			authConfig, err = config.FetchAuthConfig(opts.ServerAddress)
			if err != nil {
				return fmt.Errorf("failed to fetch auth config: %w", err)
			}
			secret = &secretstore2.HubSecret{
				AuthConfig: &secretstore2.AuthConfig{
					ClientId:           authConfig.ClientId,
					ProductId:          authConfig.IdpProductId,
					IdpFrontendAddress: authConfig.IdpFrontendAddress,
					IdpBackendAddress:  authConfig.IdpBackendAddress,
					IdpIssuerAddress:   authConfig.IdpIssuerAddress,
					HubBackendAddress:  authConfig.HubBackendAddress,
				},
			}
		}

		if secret == nil {
			return fmt.Errorf("failed to init auth config and secrets")
		}

		ctx = ctxUtils.SetCurrentHubSecretForContext(ctx, secret)
		if secret.IdpIssuerAddress == "" {
			return fmt.Errorf("issuer address is empty")
		}

		idpClient := idp.NewClient(secret.IdpIssuerAddress)
		ctx = ctxUtils.SetIdpClientForContext(ctx, idpClient)

		ctx = ctxUtils.SetCurrentServerAddressForContext(ctx, opts.ServerAddress)

		cmd.SetContext(ctx)

		return nil
	}

	cmd.AddCommand(
		login.NewCommand(opts),
		logout.NewCommand(opts),
		push.NewCommand(opts),
		pull.NewCommand(),
	)

	return cmd
}
