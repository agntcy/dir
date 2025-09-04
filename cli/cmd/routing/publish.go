// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish <cid>",
	Short: "Publish record to the network for discovery",
	Long: `Publish a record to the network to allow content discovery by other peers.

This command announces a record that is already stored locally to the distributed
network, making it discoverable by other peers through the DHT.

The record must already exist in local storage (use 'dirctl push' first if needed).

Key Features:
- Network announcement: Makes record discoverable by other peers
- Local storage: Stores record in local routing index
- DHT announcement: Announces record and labels to distributed network
- Background retry: Failed announcements are retried automatically

Usage examples:

1. Publish a record to the network:
   dirctl routing publish <cid>

Note: The record must already be pushed to storage before publishing.
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPublishCommand(cmd, args[0])
	},
}

func runPublishCommand(cmd *cobra.Command, cid string) error {
	// Get the client from the context
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
	if err := c.Publish(cmd.Context(), &routingv1.PublishRequest{
		Request: &routingv1.PublishRequest_RecordRefs{
			RecordRefs: &routingv1.RecordRefs{
				Refs: []*corev1.RecordRef{recordRef},
			},
		},
	}); err != nil {
		if strings.Contains(err.Error(), "failed to announce object") {
			return errors.New("failed to announce object, it will be retried in the background on the API server")
		}

		return fmt.Errorf("failed to publish: %w", err)
	}

	// Success
	presenter.Printf(cmd, "Successfully published!\n")
	presenter.Printf(cmd, "Record is now discoverable by other peers.\n")
	presenter.Printf(cmd, "Use 'dirctl routing search' from other peers to find this record.\n")

	return nil
}
