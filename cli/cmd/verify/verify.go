// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/cli/presenter"
	utils "github.com/agntcy/dir/cli/util/agent"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
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

	dirctl verify record.json --signature signature.json

2. Verify a record from standard input:

	dirctl pull <digest> | dirctl verify --stdin --signature signature.json
	# TODO Update to pull record and signature from store

`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			recordPath    string
		)

		if len(args) > 1 {
			return errors.New("only one file path is allowed")
		} else if len(args) == 1 {
			recordPath = args[0]
		}

		// get source
		recordSource, err := utils.GetReader(recordPath, opts.FromStdin)
		if err != nil {
			return err //nolint:wrapcheck
		}

		return runCommand(cmd, recordSource)
	},
}

// nolint:mnd
func runCommand(cmd *cobra.Command, recordSource io.ReadCloser) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Load OASF data (supports v1, v2, v3) into a Record
	record, err := corev1.LoadOASFFromReader(recordSource)
	if err != nil {
		return fmt.Errorf("failed to load OASF: %w", err)
	}

	if opts.SignaturePath == "" {
		return fmt.Errorf("signature path is not supported yet")
	}

	// Load signature from file if provided
	signatureData, err := os.ReadFile(filepath.Clean(opts.SignaturePath))
	if err != nil {
		return fmt.Errorf("failed to read signature file: %w", err)
	}

	signature := &signv1.Signature{}
	if err := json.Unmarshal(signatureData, signature); err != nil {
		return fmt.Errorf("failed to unmarshal signature: %w", err)
	}

	//nolint:nestif
	if opts.Key != "" {
		// Load the public key from file
		rawPubKey, err := os.ReadFile(filepath.Clean(opts.Key))
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		req := &signv1.VerifyRequest{
			Record:    record,
			Signature: signature,
			Provider: &signv1.VerifyRequestProvider{
				Provider: &signv1.VerifyRequestProvider_Key{
					Key: &signv1.VerifyWithKey{
						PublicKey: rawPubKey,
					},
				},
			},
		}

		// Verify the record using the provided key
		response, err := c.VerifyWithKey(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to verify record: %w", err)
		}

		if !response.GetSuccess() {
			return fmt.Errorf("signature verification failed: %s", response.GetErrorMessage())
		}
	} else {
		req := &signv1.VerifyRequest{
			Record:    record,
			Signature: signature,
			Provider: &signv1.VerifyRequestProvider{
				Provider: &signv1.VerifyRequestProvider_Oidc{
					Oidc: &signv1.VerifyWithOIDC{
						ExpectedIssuer: opts.OIDCIssuer,
						ExpectedSigner: opts.OIDCIdentity,
					},
				},
			},
		}

		// Verify the record using the OIDC provider
		response, err := c.VerifyWithOIDC(cmd.Context(), req)
		if err != nil {
			return fmt.Errorf("failed to verify record: %w", err)
		}

		if !response.GetSuccess() {
			return fmt.Errorf("signature verification failed: %s", response.GetErrorMessage())
		}
	}

	// Print success message
	presenter.Print(cmd, "Record signature verified successfully!")

	return nil
}
