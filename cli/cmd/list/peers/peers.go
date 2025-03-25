// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package peers

import (
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routetypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "peers",
	Short: "Get the list of peer addresses holding specific agents",
	Long: `Usage example:

	# Get the list of peer addresses holding specific agents, ie. find the location of data.
	dir list peers --digest <digest>

`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return runCommand(cmd)
	},
}

func runCommand(cmd *cobra.Command) error {
	// Get the client from the context.
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Lookup from digest
	isLocal, _ := cmd.Flags().GetBool("local")
	items, err := c.List(cmd.Context(), &routetypes.ListRequest{
		Local: &isLocal,
		Record: &coretypes.ObjectRef{
			Digest: opts.Digest,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list peers: %w", err)
	}

	for item := range items {
		presenter.Printf(cmd,
			"Peer: %v | Labels: %v | Annotations: %v | Digest: %v\n",
			item.GetPeer().GetId(),
			item.GetLabels(),
			item.GetRecord().GetAnnotations(),
			item.GetRecord().GetDigest(),
		)
	}

	return nil
}
