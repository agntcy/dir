// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package unpublish

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "unpublish",
	Short: "Unpublish record from the network",
	Long: `Unpublish the data from your local or rest of the network to disallow content discovery.
This command only works for the objects that are available in the store.

Usage examples:

1. Unpublish the data to the local data store:

	dirctl unpublish <cid>

2. Unpublish the data across the network:

  	dirctl unpublish <cid> --network

`,
	RunE: func(cmd *cobra.Command, args []string) error { //nolint:gocritic
		if len(args) != 1 {
			return errors.New("cid is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

func runCommand(cmd *cobra.Command, cid string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Create RecordRef from cid
	recordRef := &corev1.RecordRef{
		Cid: cid,
	}

	// Lookup metadata to verify record exists
	_, err := c.Lookup(cmd.Context(), recordRef)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	presenter.Printf(cmd, "Unpublishing record with CID: %s\n", recordRef.GetCid())

	if err := c.Unpublish(cmd.Context(), recordRef); err != nil {
		return fmt.Errorf("failed to unpublish: %w", err)
	}

	// Success
	presenter.Printf(cmd, "Successfully unpublished!\n")

	if opts.Network {
		presenter.Printf(cmd, "It may take some time for the record to be unpublished across the network.\n")
	}

	return nil
}
