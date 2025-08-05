// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/client"
	"github.com/spf13/cobra"
)

//nolint:mnd
var Command = &cobra.Command{
	Use:   "verify",
	Short: "Verify record signature against identity-based OIDC or key-based signing",
	Long: `This command verifies the record signature against
identity-based OIDC or key-based signing process.

Usage examples:

1. Verify a record from file:

	dirctl verify <record-cid>

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var recordRef string
		if len(args) > 1 {
			return errors.New("one argument is allowed")
		} else if len(args) == 1 {
			recordRef = args[0]
		}

		return runCommand(cmd, recordRef)
	},
}

// nolint:mnd
func runCommand(cmd *cobra.Command, recordRef string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	var err error

	switch {
	case opts.Key != "":
		err = verifyWithKey(cmd.Context(), c, recordRef, opts.Key)
	case opts.OIDC:
		err = verifyWithOIDC(cmd.Context(), c, recordRef, opts.OIDCIssuer, opts.OIDCIdentity)
	default:
		err = verifyWithZot(cmd.Context(), c, recordRef)
	}

	if err != nil {
		return err
	}

	// Print success message
	presenter.Print(cmd, "Record signature verified successfully!")

	return nil
}

// verifyWithKey performs key-based verification.
func verifyWithKey(ctx context.Context, c *client.Client, recordRef, keyPath string) error {
	// Load the public key from file
	rawPubKey, err := os.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	response, err := c.VerifyWithKey(ctx, &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordRef,
		},
		Provider: &signv1.VerifyRequestProvider{
			Provider: &signv1.VerifyRequestProvider_Key{
				Key: &signv1.VerifyWithKey{
					PublicKey: rawPubKey,
				},
			},
		},
	})

	return handleVerifyResponse(response, err, "verify record with key")
}

// verifyWithOIDC performs OIDC-based verification.
func verifyWithOIDC(ctx context.Context, c *client.Client, recordRef, expectedIssuer, expectedSigner string) error {
	response, err := c.VerifyWithOIDC(ctx, &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordRef,
		},
		Provider: &signv1.VerifyRequestProvider{
			Provider: &signv1.VerifyRequestProvider_Oidc{
				Oidc: &signv1.VerifyWithOIDC{
					ExpectedIssuer: expectedIssuer,
					ExpectedSigner: expectedSigner,
				},
			},
		},
	})

	return handleVerifyResponse(response, err, "verify record with OIDC")
}

// verifyWithZot performs Zot-based verification.
func verifyWithZot(ctx context.Context, c *client.Client, recordRef string) error {
	response, err := c.VerifyWithZot(ctx, &signv1.VerifyRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordRef,
		},
	})

	return handleVerifyResponse(response, err, "verify record with Zot")
}

// handleVerifyResponse processes the verification response and returns an error if verification failed.
func handleVerifyResponse(response *signv1.VerifyResponse, err error, operation string) error {
	if err != nil {
		return fmt.Errorf("failed to %s: %w", operation, err)
	}

	if !response.GetSuccess() {
		return fmt.Errorf("signature verification failed: %s", response.GetErrorMessage())
	}

	return nil
}
