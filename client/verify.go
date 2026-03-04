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
	"github.com/agntcy/dir/utils/verify"
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

	resp, _, err := verify.Verify(ctx, req, c)

	return resp, err //nolint:wrapcheck
}

// PullSignatures implements verify.Fetcher.
func (c *Client) PullSignatures(ctx context.Context, recordRef *corev1.RecordRef) ([]verify.SigWithDigest, error) {
	referrerType := corev1.SignatureReferrerType

	respCh, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	var out []verify.SigWithDigest

	for resp := range respCh {
		if resp.GetReferrer() == nil {
			continue
		}

		ref := resp.GetReferrer()

		sig := &signv1.Signature{}
		if err := sig.UnmarshalReferrer(ref); err != nil {
			logger.Debug("Failed to unmarshal signature referrer", "error", err)

			continue
		}

		out = append(out, verify.SigWithDigest{
			Digest: verify.ReferrerDigest(ref),
			Sig:    sig,
		})
	}

	return out, nil
}

// PullPublicKeys implements verify.Fetcher.
func (c *Client) PullPublicKeys(ctx context.Context, recordRef *corev1.RecordRef) ([]string, error) {
	referrerType := corev1.PublicKeyReferrerType

	respCh, err := c.PullReferrer(ctx, &storev1.PullReferrerRequest{
		RecordRef:    recordRef,
		ReferrerType: &referrerType,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull referrer stream: %w", err)
	}

	var publicKeys []string

	for resp := range respCh {
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
