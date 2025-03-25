// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agents

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
	Use:   "agents",
	Short: "List different type of objects on the network",
	Long: `Usage example:

	# Get a list of agent digests across the network that can satisfy the query.
	dirctl list agents --query <query>

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

	isLocal, _ := cmd.Flags().GetBool("local")

	maxHops := uint32(10) //nolint:mnd

	items, err := c.List(cmd.Context(), &routetypes.ListRequest{
		Labels:  strings.Split(opts.Query, ","),
		MaxHops: &maxHops,
		Local:   &isLocal,
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
