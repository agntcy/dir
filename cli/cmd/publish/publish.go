// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	"github.com/agntcy/dir/cli/util"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "publish",
	Short: "Publish compiled agent model to the network, allowing content discovery",
	Long: `Usage example:

   	# Publish the data only to the local routing table.
    dirctl publish <digest>

	# Publish the data across the network.
  	# NOTE: It is not guaranteed that this will succeed.
  	dirctl publish <digest> --network

`,
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		if len(args) != 1 {
			return errors.New("digest is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, digest string) error {
	// Get the client from the context.
	c, ok := util.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Lookup from digest
	meta, err := c.Lookup(cmd.Context(), &coretypes.ObjectRef{
		Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest: digest,
	})
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	presenter.Printf(cmd, "Publishing agent: %v\n", meta)

	// Start publishing
	if err := c.Publish(cmd.Context(), meta, opts.Network); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	// Success
	presenter.Printf(cmd, "Successfully published!\n")

	if opts.Network {
		presenter.Printf(cmd, "It may take some time for the agent to be propagated and discoverable across the network.\n")
	}

	return nil
}
