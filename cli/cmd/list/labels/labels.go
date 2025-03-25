// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	"errors"
	"fmt"
	"strings"

	routetypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "labels",
	Short: "List the labels across the network",
	Long: `Usage example:

	# Get a list of labels that across the network, ie. find the type of data.
  	# Labels are OASF (e.g. skills, locators) and Dir-specific (e.g. publisher).
   	dir list labels text text/rag
	
	# For a specific peer, run the following command.
	dir list labels text text/rag --peer <peer-id>
	

`,
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		return runCommand(cmd, args)
	},
}

func runCommand(cmd *cobra.Command, args []string) error {
	// Get the client from the context.
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Is peer set
	var peer *routetypes.Peer
	if opts.PeerId != "" {
		peer = &routetypes.Peer{
			Id: opts.PeerId,
		}
	}

	// Get a list of arguments (skills)
	var skills []string
	if len(args) > 0 {
		skills = args
	}

	// Get max hops
	maxHops := uint32(10) //nolint:mnd

	// Run extensive search
	isLocal, _ := cmd.Flags().GetBool("local")

	items, err := c.List(cmd.Context(), &routetypes.ListRequest{
		Peer:    peer,
		Labels:  skills,
		MaxHops: &maxHops,
		Local:   &isLocal,
	})
	if err != nil {
		return fmt.Errorf("failed to list peers: %w", err)
	}

	for item := range items {
		// in case we have statistics, we skip printing the item
		if len(item.GetLabelCounts()) > 0 {
			presenter.Printf(cmd, "Statistic for Peer %s\n", item.GetPeer())
			presenter.Printf(cmd, "%+v\n", item.GetLabelCounts())

			continue
		}

		// print the item
		// FIXME: this can panic if we dont return full values
		presenter.Printf(cmd,
			"Peer %v | Digest: %v | Labels: %v | Metadata: %v\n",
			item.GetPeer().GetId(),
			item.GetRecord().GetDigest(),
			strings.Join(item.GetLabels(), ", "),
			item.GetRecord().GetAnnotations(),
		)
	}

	return nil
}
