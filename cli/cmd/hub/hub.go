package hub

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand(hub Hub) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hub",
		Short: "CLI tool to interact with Agent Hub implementation",
		Run: func(cmd *cobra.Command, args []string) {
			err := hub.Run(cmd.Context(), args)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), err.Error())
			}
		},
		DisableFlagParsing: true,
	}
	return cmd
}
