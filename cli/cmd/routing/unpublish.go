// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package routing

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var unpublishCmd = &cobra.Command{
	Use:   "unpublish <cid>",
	Short: "Unpublish record from the network",
	Long: `Unpublish a record from the network to stop content discovery by other peers.

This command removes a record's network announcements, making it no longer
discoverable by other peers through the DHT. The record remains in local storage.

Key Features:
- Network removal: Removes record from distributed discovery
- Local cleanup: Removes record from local routing index
- DHT cleanup: Removes record and label announcements from network
- Immediate effect: Record becomes undiscoverable by other peers

Usage examples:

1. Unpublish a record from the network:
   dirctl routing unpublish <cid>

2. Output formats:
   # Unpublish with JSON confirmation
   dirctl routing unpublish <cid> --output json
   
   # Unpublish with raw output for scripting
   dirctl routing unpublish <cid> --output raw

Note: This only removes network announcements. Use 'dirctl delete' to remove the record entirely.
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUnpublishCommand(cmd, args[0])
	},
}

func runUnpublishCommand(cmd *cobra.Command, cid string) error {
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

	// Start unpublishing using the same RecordRef
	if err := c.Unpublish(cmd.Context(), &routingv1.UnpublishRequest{
		Request: &routingv1.UnpublishRequest_RecordRefs{
			RecordRefs: &routingv1.RecordRefs{
				Refs: []*corev1.RecordRef{recordRef},
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to unpublish: %w", err)
	}

	// Output in the appropriate format
	result := map[string]interface{}{
		"cid":     recordRef.GetCid(),
		"status":  "unpublished",
		"message": "Record is no longer discoverable by other peers",
	}

	return presenter.PrintMessage(cmd, "Unpublish", "Successfully unpublished record", result)
}
