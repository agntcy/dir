// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ownership provides CLI commands for managing ownership claims on records.
package ownership

import (
	"errors"
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ownershipv1 "github.com/agntcy/dir/api/ownership/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

// Command is the root ownership command.
var Command = &cobra.Command{
	Use:   "ownership",
	Short: "Manage ownership claims on records",
	Long: `Commands for claiming and revoking ownership of records.

Ownership claims assert that a specific identity is the operational owner of a record.
The server enforces (when auth is enabled) that the caller's identity matches the claimed owner_id.

Usage examples:

  # Claim ownership of a record
  dirctl ownership claim --record <CID> --owner alice@acme.com

  # Search for records owned by alice
  dirctl search --owner alice@acme.com
`,
}

var claimOpts = struct {
	RecordCID string
	OwnerID   string
}{}

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Claim ownership of a record",
	Long: `Attach an ownership claim referrer to a record.

The server enforces (when authentication is enabled) that the caller's
authenticated identity matches the --owner value.

Usage examples:

  dirctl ownership claim --record <CID> --owner alice@acme.com
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if claimOpts.RecordCID == "" {
			return errors.New("--record is required")
		}

		if claimOpts.OwnerID == "" {
			return errors.New("--owner is required")
		}

		c, ok := ctxUtils.GetClientFromContext(cmd.Context())
		if !ok {
			return errors.New("failed to get client from context")
		}

		claim := &ownershipv1.OwnershipClaim{
			OwnerId:   claimOpts.OwnerID,
			ClaimedAt: time.Now().UTC().Format(time.RFC3339),
		}

		ref, err := claim.MarshalReferrer()
		if err != nil {
			return fmt.Errorf("failed to marshal ownership claim: %w", err)
		}

		resp, err := c.PushReferrer(cmd.Context(), &storev1.PushReferrerRequest{
			RecordRef: &corev1.RecordRef{Cid: claimOpts.RecordCID},
			Type:      ref.GetType(),
			Data:      ref.GetData(),
		})
		if err != nil {
			return fmt.Errorf("failed to push ownership claim: %w", err)
		}

		if !resp.GetSuccess() {
			return fmt.Errorf("server rejected ownership claim: %s", resp.GetErrorMessage())
		}

		return presenter.PrintMessage(cmd, "ownership", "Ownership claimed for record", claimOpts.RecordCID)
	},
}

func init() {
	claimCmd.Flags().StringVar(&claimOpts.RecordCID, "record", "", "CID of the record to claim ownership of (required)")
	claimCmd.Flags().StringVar(&claimOpts.OwnerID, "owner", "", "Owner identity, e.g. alice@acme.com (required)")
	presenter.AddOutputFlags(claimCmd)

	Command.AddCommand(claimCmd)
}
