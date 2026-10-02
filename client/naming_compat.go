// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	namingv1 "github.com/agntcy/dir/api/naming/v1"
)

// ownerClaimMethod is the Method reported for a verified ownership claim.
const ownerClaimMethod = "owner-claim"

// GetVerificationInfo reports whether the record with the given CID has a
// verified owner.
//
// Deprecated: kept for the pinned github.com/agntcy/dir-mcp; use
// GetIdentityStatus.
func (c *Client) GetVerificationInfo(ctx context.Context, cid string) (*namingv1.GetVerificationInfoResponse, error) {
	status, err := c.GetIdentityStatus(ctx, cid)
	if err != nil {
		return nil, err
	}

	return verificationInfo(status), nil
}

// GetVerificationInfoByName is GetVerificationInfo for the record a name, with
// an optional version, resolves to.
//
// Deprecated: kept for the pinned github.com/agntcy/dir-mcp; use
// IdentityServiceClient.GetIdentityStatus.
func (c *Client) GetVerificationInfoByName(ctx context.Context, name, version string) (*namingv1.GetVerificationInfoResponse, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	req := &identityv1.GetIdentityStatusRequest{Name: &name}
	if version != "" {
		req.Version = &version
	}

	status, err := c.IdentityServiceClient.GetIdentityStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity status: %w", err)
	}

	return verificationInfo(status), nil
}

// verificationInfo projects the ownership claim result onto the legacy result.
func verificationInfo(status *identityv1.GetIdentityStatusResponse) *namingv1.GetVerificationInfoResponse {
	owner := status.GetOwner()
	if owner == nil {
		return &namingv1.GetVerificationInfoResponse{ErrorMessage: "no verification found"}
	}

	if owner.GetStatus() != identityv1.ClaimVerificationStatus_CLAIM_VERIFICATION_STATUS_VERIFIED {
		msg := owner.GetError()
		if msg == "" {
			msg = "verification failed"
		}

		return &namingv1.GetVerificationInfoResponse{ErrorMessage: msg}
	}

	return &namingv1.GetVerificationInfoResponse{
		Verified: true,
		Verification: &namingv1.Verification{
			Domain: &namingv1.DomainVerification{
				Domain:     owner.GetSubject(),
				Method:     ownerClaimMethod,
				VerifiedAt: owner.GetVerifiedAt(),
			},
		},
	}
}
