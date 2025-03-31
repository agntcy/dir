package hub

import (
	"github.com/spf13/cobra"

	"github.com/agntcy/dir/cli/cmd/hub/login"
	"github.com/agntcy/dir/cli/options"
)

func NewHubCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "hub",
		Short: "Manage the Phoenix SaaS hub",
	}

	opts := options.NewOptions()
	opts.Register(cmd)

	cmd.AddCommand(
		login.NewLoginCommand(opts),
	)

	return cmd
}
