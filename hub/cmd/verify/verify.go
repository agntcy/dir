// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0
package verify

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/presenter"
	hubClient "github.com/agntcy/dir/hub/client/hub"
	hubOptions "github.com/agntcy/dir/hub/cmd/options"
	"github.com/agntcy/dir/hub/service"
	"github.com/agntcy/dir/hub/sessionstore"
	authUtils "github.com/agntcy/dir/hub/utils/auth"
	cosignutils "github.com/agntcy/dir/utils/cosign"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/spf13/cobra"
)

var hubOpts *hubOptions.HubOptions

func NewCommand(hubOptions *hubOptions.HubOptions) *cobra.Command {
	hubOpts = hubOptions

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify record signature against identity-based OIDC or key-based signing",
		Long: `This command verifies the record signature against
identity-based OIDC or key-based signing process.

Usage examples:

1. Verify a record from file:

	dirctl hub verify <record-cid>
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("you must specify the recordCID")
			}

			recordCID := args[0]

			currentSession, err := authUtils.GetOrCreateSession(cmd, hubOpts.ServerAddress, hubOpts.APIKeyFile, false)
			if err != nil {
				return fmt.Errorf("failed to get or create session: %w", err)
			}

			hc, err := hubClient.New(currentSession.HubBackendAddress)
			if err != nil {
				return fmt.Errorf("failed to create hub client: %w", err)
			}

			trusted, err := verify(cmd.Context(), hc, currentSession, recordCID)
			if err != nil {
				return fmt.Errorf("failed to verify record: %w", err)
			}

			status := "untrusted"
			if trusted {
				status = "trusted"
			}

			return presenter.PrintMessage(cmd, "signature", "Record signature is", status)
		},
	}

	return cmd
}

func verify(ctx context.Context, hc hubClient.Client, session *sessionstore.HubSession, recordCID string) (bool, error) {
	// Generate the expected payload for this record CID
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	expectedPayload, err := cosignutils.GeneratePayload(digest.String())
	if err != nil {
		return false, fmt.Errorf("failed to generate expected payload: %w", err)
	}

	signatures, err := service.GetRecordSignatures(ctx, hc, recordCID, session)
	if err != nil {
		return false, fmt.Errorf("failed to get signatures: %w", err)
	}

	if len(signatures) == 0 {
		return false, nil
	}

	// Compare all public keys with all signatures
	for _, sig := range signatures {
		verifier, err := sigs.LoadPublicKeyRaw([]byte(sig.GetPublicKey()), crypto.SHA256)
		if err != nil {
			// This is an invalid public key, let's check the next signature
			continue
		}

		signatureBytes, err := base64.StdEncoding.DecodeString(sig.GetSignature())
		if err != nil {
			// If decoding fails, assume it's already raw bytes
			signatureBytes = []byte(sig.GetSignature())
		}

		err = verifier.VerifySignature(bytes.NewReader(signatureBytes), bytes.NewReader(expectedPayload))
		if err != nil {
			// Verification failed for this combination, try the next one
			continue
		}

		// If the signature is verified against this public key, return true
		return true, nil
	}

	return false, nil
}
