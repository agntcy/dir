package hub

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/cmd/hub/login"
	"github.com/agntcy/dir/cli/cmd/hub/logout"
	"github.com/agntcy/dir/cli/options"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
)

func NewHubCommand() *cobra.Command {
	opts := options.NewHubOptions()

	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Phoenix SaaS hub",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			opts.Complete()
			if err := opts.CheckError(); err != nil {
				return err
			}

			secretStore, ok := ctxUtils.GetSecretStoreFromContext(cmd.Context())
			if !ok {
				return fmt.Errorf("failed to get secret store from context")
			}

			secret, err := secretStore.GetHubSecret(opts.ServerAddress)
			if err != nil {
				return nil
			}

			newCtx := ctxUtils.SetCurrentHubSecretForContext(cmd.Context(), secret)
			cmd.SetContext(newCtx)

			return nil
		},
		TraverseChildren: true,
	}

	opts.Register(cmd)

	cmd.AddCommand(
		login.NewCommand(opts),
		logout.NewCommand(opts),
	)

	return cmd
}
