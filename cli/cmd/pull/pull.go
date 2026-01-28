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
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

var Command = &cobra.Command{
	Use:   "pull <cid-or-name[:version][@digest]>",
	Short: "Pull record from Directory server",
	Long: `This command pulls the record from Directory API. The data can be validated against its hash, as
the returned object is content-addressable.

You can pull by CID or by name. The command auto-detects whether the argument is a CID or a name:
- If it's a valid CID (e.g., bafyrei...), it pulls directly by CID
- Otherwise, it resolves the name to a CID and pulls it

When pulling by name without a version, the most recently created version is used.

For hash-verified pulls, append @digest to verify the resolved record matches the expected CID:
- name@digest  - verify latest version matches the digest
- name:version@digest - verify specific version matches the digest

Usage examples:

1. Pull by CID:

	dirctl pull bafyreib...

2. Pull by name (latest version):

	dirctl pull cisco.com/marketing-agent
	dirctl pull https://cisco.com/marketing-agent

3. Pull by name with specific version:

	dirctl pull cisco.com/marketing-agent:v1.0.0
	dirctl pull https://cisco.com/marketing-agent:v2.1.0

4. Pull with hash verification:

	dirctl pull cisco.com/marketing-agent@bafyreib...
	dirctl pull cisco.com/marketing-agent:v1.0.0@bafyreib...

5. Pull with public key:

	dirctl pull <cid-or-name> --public-key

6. Pull with signature:

	dirctl pull <cid-or-name> --signature

7. Output formats:

	# Get record as JSON
	dirctl pull <cid-or-name> --output json
	
	# Get record with public key as JSON
	dirctl pull <cid-or-name> --public-key --output json
	
	# Get raw record data for piping
	dirctl pull <cid-or-name> --output raw > record.json
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("cid or name is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

//nolint:cyclop,gocognit
func runCommand(cmd *cobra.Command, input string) error {
	// Get the client from the context.
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Resolve the input to a CID
	recordCID, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return err
	}

	// Fetch record from store
	record, err := c.Pull(cmd.Context(), &corev1.RecordRef{
		Cid: recordCID,
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
				Cid: recordCID,
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
				Cid: recordCID,
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
	structuredData := map[string]any{
		"record": map[string]any{
			"data": record.GetData(),
		},
	}

	// Add public keys if any
	if len(publicKeys) > 0 {
		publicKeyData := make([]map[string]any, len(publicKeys))
		for i, pk := range publicKeys {
			publicKeyData[i] = map[string]any{
				"key": pk.GetKey(),
			}
		}

		structuredData["publicKeys"] = publicKeyData
	}

	// Add signatures if any
	if len(signatures) > 0 {
		signatureData := make([]map[string]any, len(signatures))
		for i, sig := range signatures {
			signatureData[i] = map[string]any{
				"signature": sig.GetSignature(),
			}
		}

		structuredData["signatures"] = signatureData
	}

	// Output the structured data
	return presenter.PrintMessage(cmd, "record", "Record data with keys and signatures", structuredData)
}
