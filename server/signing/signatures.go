// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
)

// PushSignature attaches a signature to a record using cosign.
func (s *sign) PushSignature(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) error {
	logger.Debug("Pushing signature", "recordCID", recordCID)

	if s.ociConfig == nil {
		return errors.New("OCI configuration not set for signing service")
	}

	// Decode the signature from the referrer
	signature := &signv1.Signature{}
	if err := signature.UnmarshalReferrer(referrer); err != nil {
		return fmt.Errorf("failed to decode signature from referrer: %w", err)
	}

	if recordCID == "" {
		return errors.New("record CID is required")
	}

	// Construct the OCI image reference for the record
	imageRef := s.constructImageReference(recordCID)

	// Prepare options for attaching signature
	attachOpts := &cosign.AttachSignatureOptions{
		ImageRef:  imageRef,
		Signature: signature.GetSignature(),
		Payload:   signature.GetAnnotations()["payload"],
		Username:  s.ociConfig.Username,
		Password:  s.ociConfig.Password,
	}

	// Attach signature using cosign
	if err := cosign.AttachSignature(ctx, attachOpts); err != nil {
		return fmt.Errorf("failed to attach signature with cosign: %w", err)
	}

	logger.Debug("Signature attached successfully using cosign", "recordCID", recordCID)

	return nil
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
