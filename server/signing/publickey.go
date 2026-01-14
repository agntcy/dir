// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/zot"
)

// UploadPublicKey uploads a public key to Zot's cosign extension for signature verification.
// This is only needed for Zot registries - other registries use OCI referrers only.
func (s *sign) UploadPublicKey(ctx context.Context, referrer *corev1.RecordReferrer) error {
	logger.Debug("Uploading public key for signature verification")

	if s.zotConfig == nil {
		return errors.New("zot configuration not set for signing service")
	}

	// Decode the public key from the referrer
	pk := &signv1.PublicKey{}
	if err := pk.UnmarshalReferrer(referrer); err != nil {
		return fmt.Errorf("failed to get public key from referrer: %w", err)
	}

	publicKey := pk.GetKey()
	if publicKey == "" {
		return errors.New("public key is required")
	}

	// Upload the public key to zot for signature verification
	// This enables zot to mark this signature as "trusted" in verification queries
	uploadOpts := &zot.UploadPublicKeyOptions{
		Config:    s.zotConfig,
		PublicKey: publicKey,
	}

	if err := zot.UploadPublicKey(ctx, uploadOpts); err != nil {
		return fmt.Errorf("failed to upload public key to zot for verification: %w", err)
	}

	logger.Debug("Successfully uploaded public key for verification")

	return nil
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
