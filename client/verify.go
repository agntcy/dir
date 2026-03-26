// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/verify"
)

// Verify verifies signatures for a record.
// When from_server is true, the result is the server's cached verification;
// when false, verification is performed locally.
func (c *Client) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	if req.GetFromServer() {
		if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
			return nil, fmt.Errorf("record CID is required")
		}

		resp, err := c.SignServiceClient.Verify(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("verify: %w", err)
		}

		return resp, nil
	}

	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, fmt.Errorf("record CID is required")
	}

	_, err := c.Lookup(ctx, req.GetRecordRef())
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			errMsg := "record not found"

			// # NOTE: The returned error can be nil as the failure is indicated by the error message as a reason.
			//nolint:nilerr
			return &signv1.VerifyResponse{
				Success:      false,
				ErrorMessage: &errMsg,
			}, nil
		}

		return nil, fmt.Errorf("failed to lookup record: %w", err)
	}

	resp, _, err := verify.VerifyWithFetcher(ctx, req, c)

	return resp, err //nolint:wrapcheck
}

// PullSignatures fetches all signature referrers for a record.
func (c *Client) PullSignatures(ctx context.Context, recordRef *corev1.RecordRef) ([]*signv1.Signature, error) {
	referrerType := corev1.SignatureReferrerType

	responses, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	var signatures []*signv1.Signature

	for _, resp := range responses {
		if resp.GetReferrer() == nil {
			continue
		}

		sig := &signv1.Signature{}
		if err := sig.UnmarshalReferrer(resp.GetReferrer()); err != nil {
			logger.Debug("Failed to unmarshal signature referrer", "error", err)

			continue
		}

		signatures = append(signatures, sig)
	}

	return signatures, nil
}

// PullPublicKeys fetches all public key referrers for a record.
func (c *Client) PullPublicKeys(ctx context.Context, recordRef *corev1.RecordRef) ([]string, error) {
	referrerType := corev1.PublicKeyReferrerType

	responses, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	var publicKeys []string

	for _, resp := range responses {
		if resp.GetReferrer() == nil {
			continue
		}

		pk := &signv1.PublicKey{}
		if err := pk.UnmarshalReferrer(resp.GetReferrer()); err != nil {
			logger.Debug("Failed to unmarshal public key referrer", "error", err)

			continue
		}

		if key := pk.GetKey(); key != "" {
			publicKeys = append(publicKeys, key)
		}
	}

	return publicKeys, nil
}
