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
	Use:   "unpublish <cid> [cid...]",
	Short: "Unpublish record from the network",
	Long: `Unpublish a record from the network to stop content discovery by other peers.

This command removes a record's network announcements, making it no longer
discoverable by other peers through the DHT. Records remain in local storage.

Key Features:
- Network removal: Removes record from distributed discovery
- Local cleanup: Removes record from local routing index
- DHT cleanup: Removes record and label announcements from network
- Immediate effect: Record becomes undiscoverable by other peers
- Batch unpublication: Submit multiple CIDs in one request

Usage examples:

1. Unpublish a record from the network:
   dirctl routing unpublish <cid>

2. Unpublish multiple records in a single request:
   dirctl routing unpublish <cid1> <cid2> <cid3>

3. Unpublish records from stdin (JSON array or line-delimited CIDs):
   dirctl search --format cid --limit 100 --output json | dirctl routing unpublish --stdin

4. Output formats:
   # Unpublish with JSON confirmation
   dirctl routing unpublish <cid> --output json
   
   # Unpublish with raw output for scripting
   dirctl routing unpublish <cid> --output raw

Note: This only removes network announcements. Use 'dirctl delete' to remove the record entirely.
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if unpublishOpts.FromStdin {
			return cobra.MaximumNArgs(0)(cmd, args)
		}

		return cobra.MinimumNArgs(1)(cmd, args)
	},
	RunE: runUnpublishCommand,
}

var unpublishOpts struct {
	FromStdin bool
}

func init() {
	unpublishCmd.Flags().BoolVar(&unpublishOpts.FromStdin, "stdin", false,
		"Read CIDs from standard input. Supports JSON array output from 'dirctl search --output json' and line-delimited CIDs.")
}

func runUnpublishCommand(cmd *cobra.Command, args []string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	cids := append([]string{}, args...)

	if unpublishOpts.FromStdin {
		stdinCIDs, err := readCIDsFromStdin(cmd.InOrStdin())
		if err != nil {
			return fmt.Errorf("failed to read CIDs from stdin: %w", err)
		}

		cids = append(cids, stdinCIDs...)
	}

	cids = deduplicateCIDs(cids)
	if len(cids) == 0 {
		return errors.New("at least one CID is required (pass arguments or use --stdin)")
	}

	recordRefs := make([]*corev1.RecordRef, 0, len(cids))
	for _, cid := range cids {
		recordRefs = append(recordRefs, &corev1.RecordRef{Cid: cid})
	}

	// Lookup metadata to verify records exist
	_, err := c.LookupBatch(cmd.Context(), recordRefs)
	if err != nil {
		return fmt.Errorf("failed to lookup: %w", err)
	}

	// Start unpublishing using record references
	if err := c.Unpublish(cmd.Context(), &routingv1.UnpublishRequest{
		Request: &routingv1.UnpublishRequest_RecordRefs{
			RecordRefs: &routingv1.RecordRefs{
				Refs: recordRefs,
			},
		},
	}); err != nil {
		return fmt.Errorf("failed to unpublish: %w", err)
	}

	// Output in the appropriate format
	result := map[string]any{
		"count":   len(recordRefs),
		"cids":    cids,
		"status":  "Successfully submitted unpublication request",
		"message": "Records will not be discoverable by other peers after the unpublication service processes the request.",
	}

	if len(cids) == 1 {
		result["cid"] = cids[0]
	}

	return presenter.PrintMessage(cmd, "Unpublish", "Successfully unpublished record", result)
}
