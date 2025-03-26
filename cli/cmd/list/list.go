// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package list

import (
	"errors"

	"github.com/agntcy/dir/cli/cmd/list/info"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "list",
	Short: "Search for published records across the network",
	Long: `Usage example:

	# List all peers that are providing a specific object
   	dir list --digest <digest>
	
	# List published records on the local node
   	dir list "/skills/Text Completion [10201]"

	# List published records on a specific peer
   	dir list "/skills/Text Completion [10201]" --peer <peer-id>
	
	# List published records by the whole network.
	# NOTE: This starts a DHT walk, so it may take a while.
	# NOTE: Results are not guaranteed to be complete and up-to-date.
   	dir list "/skills/Text Completion [10201]" --network
	
	#
	# NOTES:
	#
	# 	To search for specific records across the network, you must specify 
	# 	matching labels passed as arguments. The matching is performed using
	# 	exact set-membership rule, meaning that all labels must match.
	#   Network search relies on naive guesses by relying on the summary data
	#   to decide which peer to visit next. This may result in incomplete or
	#   outdated results.
   	#

`,
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		return runCommand(cmd, args)
	},
}

func runCommand(cmd *cobra.Command, labels []string) error {
	// Get the client from the context.
	client, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// if we request --digest, ignore everything else
	if opts.Digest != "" {
		return listDigest(cmd, client, opts.Digest)
	}

	// validate that we have labels for all the flows below
	if len(labels) == 0 {
		return errors.New("no labels specified")
	}

	if opts.Network {
		return listNetwork(cmd, client, labels)
	}

	return listPeer(cmd, client, opts.PeerID, labels)
}

func init() {
	// Common flags for all list subcommands
	// TODO: enable the commands below and wire them in where needed.
	//
	// cmd.Flags().Int("max-hops", 0, "Limit the number of routing hops when traversing the network")
	// cmd.Flags().Bool("sync", false, "Sync the discovered data into our local routing table")
	// cmd.Flags().Bool("pull", false, "Pull the discovered data into our local storage layer")
	// cmd.Flags().Bool("verify", false, "Verify each received record when pulling data")
	// cmd.Flags().StringSlice("allowed", nil, "Allow-list specific peer IDs during network traversal")
	// cmd.Flags().StringSlice("blocked", nil, "Block-list specific peer IDs during network traversal")
	Command.AddCommand(info.Command)
}
