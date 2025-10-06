// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	cosignutils "github.com/agntcy/dir/utils/cosign"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
)

// Verify verifies the signature of the record.
func (c *Client) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	// Server-side verification
	response, err := c.SignServiceClient.Verify(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("server verification failed: %w", err)
	}

	if response.GetSuccess() {
		return response, nil
	}

	// Fall back to client-side verification
	logger.Info("Server verification failed, falling back to client-side verification")

	var errMsg string

	verified, err := c.verifyClientSide(ctx, req.GetRecordRef().GetCid())
	if err != nil {
		errMsg = err.Error()
	}

	return &signv1.VerifyResponse{
		Success:      verified,
		ErrorMessage: &errMsg,
	}, nil
}

// verifyClientSide performs client-side signature verification using OCI referrers.
func (c *Client) verifyClientSide(ctx context.Context, recordCID string) (bool, error) {
	logger.Debug("Starting client-side verification", "recordCID", recordCID)

	// Generate the expected payload for this record CID
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	expectedPayload, err := cosignutils.GeneratePayload(digest.String())
	if err != nil {
		return false, fmt.Errorf("failed to generate expected payload: %w", err)
	}

	// Retrieve signature from OCI referrers
	signatures, err := c.pullSignatureReferrer(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull signature referrer: %w", err)
	}

	if len(signatures) == 0 {
		return false, errors.New("no signature found in referrer responses")
	}

	// Retrieve public key from OCI referrers
	publicKeys, err := c.pullPublicKeyReferrer(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull public key referrer: %w", err)
	}

	if len(publicKeys) == 0 {
		return false, errors.New("no public key found in referrer responses")
	}

	// Compare all public keys with all signatures
	for _, publicKey := range publicKeys {
		for _, signature := range signatures {
			// Verify signature using cosign
			verifier, err := sigs.LoadPublicKeyRaw([]byte(publicKey), crypto.SHA256)
			if err != nil {
				// Skip this public key if it's invalid, try the next one
				logger.Debug("Failed to load public key, skipping", "error", err)

				continue
			}

			// Decode base64 signature if needed
			signatureBytes, err := base64.StdEncoding.DecodeString(signature.GetSignature())
			if err != nil {
				// If decoding fails, assume it's already raw bytes
				signatureBytes = []byte(signature.GetSignature())
			}

			// Verify signature against the expected payload
			err = verifier.VerifySignature(bytes.NewReader(signatureBytes), bytes.NewReader(expectedPayload))
			if err != nil {
				// Verification failed for this combination, try the next one
				logger.Debug("Signature verification failed, trying next combination", "error", err)

				continue
			}

			// If the signature is verified against this public key, return true
			return true, nil
		}
	}

	return false, nil
}

// pullSignatureReferrer retrieves the signature referrer for a record.
func (c *Client) pullSignatureReferrer(ctx context.Context, recordCID string) ([]*signv1.Signature, error) {
	signatureType := corev1.SignatureReferrerType

	resultCh, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordCID,
		},
		ReferrerType: &signatureType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull signature referrer: %w", err)
	}

	signatures := make([]*signv1.Signature, 0)

	// Get all signature responses and decode them from referrer data
	for response := range resultCh {
		referrer := response.GetReferrer()
		if referrer != nil {
			signature := &signv1.Signature{}
			if err := signature.UnmarshalReferrer(referrer); err != nil {
				logger.Error("Failed to decode signature from referrer", "error", err)

				continue
			}

			signatures = append(signatures, signature)
		}
	}

	return signatures, nil
}

// pullPublicKeyReferrer retrieves the public key referrer for a record.
func (c *Client) pullPublicKeyReferrer(ctx context.Context, recordCID string) ([]string, error) {
	publicKeyType := corev1.PublicKeyReferrerType

	resultCh, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordCID,
		},
		ReferrerType: &publicKeyType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull public key referrer: %w", err)
	}

	publicKeys := make([]string, 0)

	// Get all public key responses and extract the public key from referrer data
	for response := range resultCh {
		referrer := response.GetReferrer()
		if referrer != nil {
			publicKey := &signv1.PublicKey{}
			if err := publicKey.UnmarshalReferrer(referrer); err != nil {
				logger.Error("Failed to decode public key from referrer", "error", err)

				continue
			}

			publicKeys = append(publicKeys, publicKey.GetKey())
		}
	}

	return publicKeys, nil
}
