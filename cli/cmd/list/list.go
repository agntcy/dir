// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"fmt"

	"github.com/agntcy/dir/cli/cmd/list/agents"
	"github.com/agntcy/dir/cli/cmd/list/labels"
	"github.com/agntcy/dir/cli/cmd/list/peers"
	"github.com/agntcy/dir/cli/util"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

var clientConfig = client.DefaultConfig

var Command = &cobra.Command{
	Use:   "list",
	Short: "List different type of objects on the network",
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Set client via context for all requests
		// TODO: make client config configurable via CLI args
		c, err := client.New(client.WithConfig(&clientConfig))
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		ctx := util.SetClientForContext(cmd.Context(), c)
		cmd.SetContext(ctx)

		// Ensure all subcommands inherit the flags
		for _, subCmd := range cmd.Commands() {
			subCmd.Flags().AddFlagSet(cmd.PersistentFlags())
		}

		return nil
	},
}

func init() {
	// Common flags for all list subcommands
	for _, cmd := range []*cobra.Command{peers.Command, labels.Command, agents.Command} {
		cmd.Flags().Bool("local", false, "List data available on the local routing table")
		cmd.Flags().Int("max-hops", 0, "Limit the number of routing hops when traversing the network")
		cmd.Flags().Bool("sync", false, "Sync the discovered data into our local routing table")
		cmd.Flags().Bool("pull", false, "Pull the discovered data into our local storage layer")
		cmd.Flags().Bool("verify", false, "Verify each received record when pulling data")
		cmd.Flags().StringSlice("allowed", nil, "Allow-list specific peer IDs during network traversal")
		cmd.Flags().StringSlice("blocked", nil, "Block-list specific peer IDs during network traversal")
	}

	Command.AddCommand(peers.Command, labels.Command, agents.Command)
}
