package hub

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/cmd/hub/login"
	"github.com/agntcy/dir/cli/cmd/hub/logout"
	"github.com/agntcy/dir/cli/cmd/hub/pull"
	"github.com/agntcy/dir/cli/cmd/hub/push"
	"github.com/agntcy/dir/cli/hub/auth"
	"github.com/agntcy/dir/cli/hub/idp"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

func NewHubCommand() *cobra.Command {
	opts := options.NewHubOptions()

	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Phoenix SaaS hub",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get secret store from context")
			}

			ctx := cmd.Context()
			var idpAddress string
			secret, err := secretStore.GetHubSecret(opts.ServerAddress)
			if err == nil {
				ctx = ctxUtils.SetCurrentHubSecretForContext(ctx, secret)
				if secret.IdpAddress != "" {
					idpAddress = secret.IdpAddress
				}
			}

			if idpAddress == "" {
				authConfig, err := auth.FetchAuthConfig(opts.ServerAddress)
				if err != nil {
					return fmt.Errorf("failed to fetch auth config: %w", err)
				}
				idpAddress = authConfig.IdpAddress
			}

			idpClient := idp.NewClient(idpAddress)
			ctx = ctxUtils.SetIdpClientForContext(ctx, idpClient)

			cmd.SetContext(ctx)

			return nil
		},
		TraverseChildren: true,
	}

	cmd.AddCommand(
		login.NewCommand(opts),
		logout.NewCommand(opts),
		push.NewCommand(opts, options.NewPushOptions()),
		pull.NewCommand(),
	)

	return cmd
}
