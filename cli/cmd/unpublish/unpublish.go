// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package unpublish

import (
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "unpublish",
	Short: "Unpublish agent model from the network",
	Long: `Unpublish the data from your local or rest of the network to disallow content discovery.
This command only works for the objects that are available in the store.

Usage examples:

1. Unpublish the data to the local data store:

	dirctl unpublish <digest>

2. Unpublish the data across the network:

  	dirctl unpublish <digest> --network

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
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
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

	presenter.Printf(cmd, "Unpublishing agent with digest: %s\n", meta.GetDigest())

	if err := c.Unpublish(cmd.Context(), meta); err != nil {
		return fmt.Errorf("failed to unpublish: %w", err)
	}

	// Success
	presenter.Printf(cmd, "Successfully unpublished!\n")

	if opts.Network {
		presenter.Printf(cmd, "It may take some time for the agent to be unpublished across the network.\n")
	}

	return nil
}
