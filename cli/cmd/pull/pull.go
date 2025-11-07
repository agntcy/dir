// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package pull

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "pull",
	Short: "Pull record from Directory server",
	Long: `This command pulls the record from Directory API. The data can be validated against its hash, as
the returned object is content-addressable.

Usage examples:

1. Pull by cid and output

	dirctl pull <cid>

2. Pull by cid and output public key

	dirctl pull <cid> --public-key

3. Pull by cid and output signature

	dirctl pull <cid> --signature

4. Output formats:

	# Get record as JSON
	dirctl pull <cid> --output json
	
	# Get record with public key as JSON
	dirctl pull <cid> --public-key --output json
	
	# Get raw record data for piping
	dirctl pull <cid> --output raw > record.json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("cid is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

//nolint:cyclop,gocognit
func runCommand(cmd *cobra.Command, cid string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Fetch record from store
	record, err := c.Pull(cmd.Context(), &corev1.RecordRef{
		Cid: cid,
	})
	if err != nil {
		return fmt.Errorf("failed to pull data: %w", err)
	}

	if !opts.PublicKey && !opts.Signature {
		// Handle different output formats
		return presenter.PrintMessage(cmd, "record", "Record data", record.GetData())
	}

	publicKeys := make([]*signv1.PublicKey, 0)

	if opts.PublicKey {
		publicKeyType := corev1.PublicKeyReferrerType

		resultCh, err := c.PullReferrer(cmd.Context(), &storev1.PullReferrerRequest{
			RecordRef: &corev1.RecordRef{
				Cid: cid,
			},
			ReferrerType: &publicKeyType,
		})
		if err != nil {
			return fmt.Errorf("failed to pull public key: %w", err)
		}

		for response := range resultCh {
			publicKey := &signv1.PublicKey{}
			if err := publicKey.UnmarshalReferrer(response.GetReferrer()); err != nil {
				return fmt.Errorf("failed to decode public key from referrer: %w", err)
			}

			if publicKey.GetKey() != "" {
				publicKeys = append(publicKeys, publicKey)
			}
		}
	}

	signatures := make([]*signv1.Signature, 0)

	if opts.Signature {
		signatureType := corev1.SignatureReferrerType

		resultCh, err := c.PullReferrer(cmd.Context(), &storev1.PullReferrerRequest{
			RecordRef: &corev1.RecordRef{
				Cid: cid,
			},
			ReferrerType: &signatureType,
		})
		if err != nil {
			return fmt.Errorf("failed to pull signature: %w", err)
		}

		for response := range resultCh {
			signature := &signv1.Signature{}
			if err := signature.UnmarshalReferrer(response.GetReferrer()); err != nil {
				return fmt.Errorf("failed to decode signature from referrer: %w", err)
			}

			if signature.GetSignature() != "" {
				signatures = append(signatures, signature)
			}
		}
	}

	// Create structured data object
	structuredData := map[string]interface{}{
		"record": map[string]interface{}{
			"data": record.GetData(),
		},
	}

	// Add public keys if any
	if len(publicKeys) > 0 {
		publicKeyData := make([]map[string]interface{}, len(publicKeys))
		for i, pk := range publicKeys {
			publicKeyData[i] = map[string]interface{}{
				"key": pk.GetKey(),
			}
		}

		structuredData["publicKeys"] = publicKeyData
	}

	// Add signatures if any
	if len(signatures) > 0 {
		signatureData := make([]map[string]interface{}, len(signatures))
		for i, sig := range signatures {
			signatureData[i] = map[string]interface{}{
				"signature": sig.GetSignature(),
			}
		}

		structuredData["signatures"] = signatureData
	}

	// Output the structured data
	return presenter.PrintMessage(cmd, "record", "Record data with keys and signatures", structuredData)
}
