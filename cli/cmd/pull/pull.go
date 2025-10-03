// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pull

import (
	"encoding/json"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	referrerutils "github.com/agntcy/dir/utils/referrer"
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
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("cid is a required argument")
		}

		return runCommand(cmd, args[0])
	},
}

//nolint:cyclop
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

	// If raw format flag is set, print and exit
	if opts.FormatRaw {
		rawData, err := record.Marshal()
		if err != nil {
			return fmt.Errorf("failed to marshal record to raw format: %w", err)
		}

		presenter.Print(cmd, string(rawData))

		return nil
	}

	// Pretty-print the OASF object
	output, err := json.MarshalIndent(record.GetData(), "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal OASF object to JSON: %w", err)
	}

	presenter.Print(cmd, string(output))

	if opts.PublicKey {
		publicKeyType := referrerutils.PublicKeyArtifactMediaType

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
			publicKey, err := referrerutils.DecodePublicKeyFromReferrer(response.GetReferrer())
			if err != nil {
				return fmt.Errorf("failed to decode public key from referrer: %w", err)
			}

			if publicKey != "" {
				presenter.Println(cmd, "Public key: "+publicKey)
			}
		}
	}

	if opts.Signature {
		signatureType := referrerutils.SignatureArtifactType

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
			signature, err := referrerutils.DecodeSignatureFromReferrer(response.GetReferrer())
			if err != nil {
				return fmt.Errorf("failed to decode cosign signature from referrer: %w", err)
			}

			if signature != nil {
				presenter.Println(cmd, "Signature: "+signature.GetSignature())
			}
		}
	}

	return nil
}
