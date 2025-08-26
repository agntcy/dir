// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "publish",
	Short: "Publish record to the network, allowing content discovery",
	Long: `Publish the data to your local or rest of the network to allow content discovery.
This command only works for the objects already pushed to store.

Usage examples:

1. Publish the data to the local data store:

	dirctl publish <cid>

2. Publish the data across the network:

  	dirctl publish <cid> --network

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

	presenter.Printf(cmd, "Publishing record with CID: %s\n", recordRef.GetCid())

	// Start publishing using the same RecordRef
	if err := c.Publish(cmd.Context(), recordRef); err != nil {
		if strings.Contains(err.Error(), "failed to announce object") {
			return errors.New("failed to announce object, it will be retried in the background on the API server")
		}

		return fmt.Errorf("failed to publish: %w", err)
	}

	// Success
	presenter.Printf(cmd, "Successfully published!\n")

	if opts.Network {
		presenter.Printf(cmd, "It may take some time for the record to be propagated and discoverable across the network.\n")
	}

	return nil
}
