// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/agntcy/dir/utils/zot"
)

// Verify verifies a record signature.
// It returns true if the signature is valid and trusted.
//
// The verification strategy depends on the available backends:
//   - If Zot is configured: tries Zot's GraphQL API first, falls back to referrers
//   - Otherwise: uses OCI referrers for standalone verification
func (s *sign) Verify(ctx context.Context, recordCID string) (bool, error) {
	logger.Debug("Verifying signature", "recordCID", recordCID)

	// Try Zot verification if configured
	if s.zotConfig != nil {
		verified, err := s.verifyWithZot(ctx, recordCID)
		if err == nil && verified {
			return true, nil
		}

		if err != nil {
			logger.Debug("Zot verification failed, falling back to referrer verification",
				"recordCID", recordCID, "error", err)
		}
	}

	// Fall back to standalone verification using OCI referrers
	return s.verifyWithReferrers(ctx, recordCID)
}

// verifyWithZot queries Zot's verification API to check if a signature is valid.
func (s *sign) verifyWithZot(ctx context.Context, recordCID string) (bool, error) {
	verifyOpts := &zot.VerificationOptions{
		Config:    s.zotConfig,
		RecordCID: recordCID,
	}

	result, err := zot.Verify(ctx, verifyOpts)
	if err != nil {
		return false, fmt.Errorf("failed to verify with zot: %w", err)
	}

	// Return the trusted status (which implies signed as well)
	return result.IsTrusted, nil
}

// verifyWithReferrers performs signature verification using OCI referrers.
// This works independently of Zot by:
// 1. Retrieving signatures from OCI referrers
// 2. Retrieving public keys from OCI referrers
// 3. Using shared verification logic to find a valid signature
//
// This approach mirrors Zot's VerifyCosignSignature pattern without requiring Zot extensions.
func (s *sign) verifyWithReferrers(ctx context.Context, recordCID string) (bool, error) {
	logger.Debug("Starting signature verification with referrers", "recordCID", recordCID)

	// Generate the expected payload for this record CID
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	expectedPayload, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return false, fmt.Errorf("failed to generate expected payload: %w", err)
	}

	// Retrieve signatures from OCI referrers
	signatures, err := s.pullSignatureReferrers(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull signature referrers: %w", err)
	}

	if len(signatures) == 0 {
		logger.Debug("No signatures found in referrers", "recordCID", recordCID)

		return false, nil
	}

	logger.Debug("Retrieved signatures from referrers", "recordCID", recordCID, "count", len(signatures))

	// Retrieve public keys from OCI referrers
	publicKeys, err := s.pullPublicKeyReferrers(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull public key referrers: %w", err)
	}

	if len(publicKeys) == 0 {
		logger.Debug("No public keys found in referrers", "recordCID", recordCID)

		return false, nil
	}

	logger.Debug("Retrieved public keys from referrers", "recordCID", recordCID, "count", len(publicKeys))

	// Convert signatures to string slice for shared verification
	sigStrings := make([]string, len(signatures))
	for i, sig := range signatures {
		sigStrings[i] = sig.GetSignature()
	}

	// Use shared verification logic
	verified, err := cosign.VerifySignatures(&cosign.VerifySignaturesOptions{
		ExpectedPayload: expectedPayload,
		Signatures:      sigStrings,
		PublicKeys:      publicKeys,
	})
	if err != nil {
		return false, fmt.Errorf("failed to verify signatures: %w", err)
	}

	if verified {
		logger.Info("Signature verified successfully", "recordCID", recordCID)
	} else {
		logger.Debug("No valid signature found for any public key", "recordCID", recordCID)
	}

	return verified, nil
}

// pullSignatureReferrers retrieves signature referrers for a record from OCI registry.
func (s *sign) pullSignatureReferrers(ctx context.Context, recordCID string) ([]*signv1.Signature, error) {
	signatures := make([]*signv1.Signature, 0)

	err := s.store.WalkReferrers(ctx, recordCID, corev1.SignatureReferrerType, func(referrer *corev1.RecordReferrer) error {
		signature := &signv1.Signature{}
		if err := signature.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to decode signature from referrer", "error", err)

			return nil // Skip this referrer but continue walking
		}

		signatures = append(signatures, signature)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull signature referrers: %w", err)
	}

	return signatures, nil
}

// pullPublicKeyReferrers retrieves public key referrers for a record from OCI registry.
func (s *sign) pullPublicKeyReferrers(ctx context.Context, recordCID string) ([]string, error) {
	publicKeys := make([]string, 0)

	err := s.store.WalkReferrers(ctx, recordCID, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		publicKey := &signv1.PublicKey{}
		if err := publicKey.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to decode public key from referrer", "error", err)

			return nil // Skip this referrer but continue walking
		}

		if publicKey.GetKey() != "" {
			publicKeys = append(publicKeys, publicKey.GetKey())
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull public key referrers: %w", err)
	}

	return publicKeys, nil
}
