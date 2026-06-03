// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ownership implements the dirctl ownership subcommand.
package ownership

import (
	"fmt"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ownershipv1 "github.com/agntcy/dir/api/ownership/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

// Command is the root ownership subcommand.
var Command = &cobra.Command{
	Use:   "ownership",
	Short: "Manage ownership claims for records",
}

var claimCmd = &cobra.Command{
	Use:   "claim",
	Short: "Attach an ownership claim referrer to a record",
	RunE:  runClaim,
}

var (
	claimRecord string
	claimOwner  string
	claimKey    string
	claimCert   string
)

func init() {
	claimCmd.Flags().StringVar(&claimRecord, "record", "", "CID of the record to claim ownership of (required)")
	claimCmd.Flags().StringVar(&claimOwner, "owner", "", "Owner identity (SPIFFE ID) to claim (required)")
	claimCmd.Flags().StringVar(&claimKey, "key", "", "Path to PEM-encoded private key for signing the claim")
	claimCmd.Flags().StringVar(&claimCert, "cert", "", "Path to PEM-encoded X.509 certificate whose URI SAN must match --owner")

	_ = claimCmd.MarkFlagRequired("record")
	_ = claimCmd.MarkFlagRequired("owner")

	claimCmd.MarkFlagsRequiredTogether("key", "cert")

	Command.AddCommand(claimCmd)
}

func runClaim(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	c, ok := ctxUtils.GetClientFromContext(ctx)
	if !ok {
		return fmt.Errorf("failed to get client from context")
	}

	claim := &ownershipv1.Claim{
		OwnerId:   claimOwner,
		ClaimedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Sign the claim when key and cert are provided.
	// The signing step validates that the cert's URI SAN matches --owner before
	// touching the network, so identity mismatches are caught client-side.
	if claimKey != "" {
		if err := ownershipv1.SignClaimWithKeyFile(claim, claimKey, claimCert); err != nil {
			return fmt.Errorf("failed to sign ownership claim: %w", err)
		}
	}

	ref, err := claim.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("failed to marshal ownership claim: %w", err)
	}

	resp, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef: &corev1.RecordRef{Cid: claimRecord},
		Type:      corev1.OwnershipClaimReferrerType,
		Data:      ref.GetData(),
	})
	if err != nil {
		return fmt.Errorf("failed to push ownership claim: %w", err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("push ownership claim failed: %s", resp.GetErrorMessage())
	}

	if ownershipv1.IsSigned(claim) {
		cmd.Printf("Signed ownership claim pushed successfully for record %s\n", claimRecord)
	} else {
		cmd.Printf("Ownership claim pushed successfully for record %s\n", claimRecord)
	}

	return nil
}
