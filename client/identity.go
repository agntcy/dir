// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/jws"
)

// ClaimIdentity signs a claim that subject is the identity of the record with
// the given CID, and pushes it to the record as a referrer. Pass
// identity.WithCertificate for a subject whose proof is a certificate (SPIFFE).
//
// The claim is stored unverified: it is checked later by the reconciler, and
// GetIdentityStatus reports the outcome.
func (c *Client) ClaimIdentity(ctx context.Context, cid, subject string, signer jws.Signer, opts ...identity.SignOption) error {
	return c.claim(ctx, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, cid, subject, signer, opts...)
}

// ClaimOwnership signs a claim that subject owns the record with the given CID,
// and pushes it to the record as a referrer. See ClaimIdentity.
func (c *Client) ClaimOwnership(ctx context.Context, cid, subject string, signer jws.Signer, opts ...identity.SignOption) error {
	return c.claim(ctx, identityv1.ClaimRole_CLAIM_ROLE_OWNER, cid, subject, signer, opts...)
}

func (c *Client) claim(ctx context.Context, role identityv1.ClaimRole, cid, subject string, signer jws.Signer, opts ...identity.SignOption) error {
	if cid == "" {
		return errors.New("cid is required")
	}

	claim := &identityv1.Claim{Role: role, Subject: subject}
	if err := identity.Sign(claim, cid, signer, opts...); err != nil {
		return fmt.Errorf("failed to sign claim: %w", err)
	}

	referrer, err := claim.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("failed to marshal claim: %w", err)
	}

	resp, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef:   &corev1.RecordRef{Cid: cid},
		Type:        referrer.GetType(),
		Annotations: referrer.GetAnnotations(),
		CreatedAt:   referrer.GetCreatedAt(),
		Data:        referrer.GetData(),
	})
	if err != nil {
		return fmt.Errorf("failed to push claim: %w", err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("failed to push claim: %s", resp.GetErrorMessage())
	}

	return nil
}

// GetIdentityStatus returns the last verification result of the identity and
// ownership claims of the record with the given CID. A claim with no result yet
// is unset in the response.
func (c *Client) GetIdentityStatus(ctx context.Context, cid string) (*identityv1.GetIdentityStatusResponse, error) {
	if cid == "" {
		return nil, errors.New("cid is required")
	}

	resp, err := c.IdentityServiceClient.GetIdentityStatus(ctx, &identityv1.GetIdentityStatusRequest{Cid: &cid})
	if err != nil {
		return nil, fmt.Errorf("failed to get identity status: %w", err)
	}

	return resp, nil
}

// ResolveIdentity resolves a record name, with an optional version, to CIDs.
// Returns all matching records newest first; if version is empty, all versions.
func (c *Client) ResolveIdentity(ctx context.Context, name, version string) (*identityv1.ResolveResponse, error) {
	req := &identityv1.ResolveRequest{Name: name}
	if version != "" {
		req.Version = &version
	}

	resp, err := c.IdentityServiceClient.Resolve(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve record: %w", err)
	}

	return resp, nil
}
