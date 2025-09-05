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
	logger.Debug("Server verification failed, falling back to client-side verification")

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

	// Retrieve signature from OCI referrers
	signature, err := c.pullSignatureReferrer(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull signature referrer: %w", err)
	}

	// Get the payload from the signature annotations
	payload := signature.GetAnnotations()["payload"]

	// Retrieve public key from OCI referrers
	publicKey, err := c.pullPublicKeyReferrer(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull public key referrer: %w", err)
	}

	// Verify signature using cosign
	verifier, err := sigs.LoadPublicKeyRaw([]byte(publicKey), crypto.SHA256)
	if err != nil {
		return false, fmt.Errorf("failed to get public key: %w", err)
	}

	// Decode base64 signature if needed
	signatureBytes, err := base64.StdEncoding.DecodeString(signature.GetSignature())
	if err != nil {
		// If decoding fails, assume it's already raw bytes
		signatureBytes = []byte(signature.GetSignature())
	}

	err = verifier.VerifySignature(bytes.NewReader(signatureBytes), bytes.NewReader([]byte(payload)))
	if err != nil {
		return false, fmt.Errorf("failed to verify signature: %w", err)
	}

	return true, nil
}

// pullSignatureReferrer retrieves the signature referrer for a record.
func (c *Client) pullSignatureReferrer(ctx context.Context, recordCID string) (*signv1.Signature, error) {
	response, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordCID,
		},
		Options: &storev1.PullReferrerRequest_PullSignature{
			PullSignature: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull signature referrer: %w", err)
	}

	signature := response.GetSignature()
	if signature == nil {
		return nil, errors.New("no signature found in referrer response")
	}

	return signature, nil
}

// pullPublicKeyReferrer retrieves the public key referrer for a record.
func (c *Client) pullPublicKeyReferrer(ctx context.Context, recordCID string) (string, error) {
	response, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef: &corev1.RecordRef{
			Cid: recordCID,
		},
		Options: &storev1.PullReferrerRequest_PullPublicKey{
			PullPublicKey: true,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to pull public key referrer: %w", err)
	}

	publicKey := response.GetPublicKey()
	if publicKey == "" {
		return "", errors.New("no public key found in referrer response")
	}

	return publicKey, nil
}
