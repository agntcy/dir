// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/jws"
)

// ClaimIdentity signs a claim that the record with the given CID has the
// identity declared in its "agntcy.dir/identity" annotation, and pushes it to the
// record as a referrer. It returns the claim that was pushed. Pass
// identity.WithCertificate for a spiffe:// subject, whose proof is a certificate.
//
// The claim is stored unverified: it is checked later by the reconciler, and
// GetIdentityStatus reports the outcome.
func (c *Client) ClaimIdentity(ctx context.Context, cid string, signer jws.Signer, opts ...identity.SignOption) (*identityv1.Claim, error) {
	return c.claim(ctx, identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, cid, signer, opts...)
}

// ClaimOwnership signs a claim that the record with the given CID is owned by
// the owner declared in its "agntcy.dir/owner" annotation, and pushes it to the
// record as a referrer. See ClaimIdentity.
func (c *Client) ClaimOwnership(ctx context.Context, cid string, signer jws.Signer, opts ...identity.SignOption) (*identityv1.Claim, error) {
	return c.claim(ctx, identityv1.ClaimRole_CLAIM_ROLE_OWNER, cid, signer, opts...)
}

func (c *Client) claim(ctx context.Context, role identityv1.ClaimRole, cid string, signer jws.Signer, opts ...identity.SignOption) (*identityv1.Claim, error) {
	if cid == "" {
		return nil, errors.New("cid is required")
	}

	record, err := c.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		return nil, fmt.Errorf("failed to pull record: %w", err)
	}

	annotation, subject := corev1.AnnotationKeyIdentity, record.GetIdentity()
	if role == identityv1.ClaimRole_CLAIM_ROLE_OWNER {
		annotation, subject = corev1.AnnotationKeyOwner, record.GetOwner()
	}

	if subject == "" {
		return nil, fmt.Errorf("record %s has no %q annotation to claim", cid, annotation)
	}

	claim := &identityv1.Claim{Role: role, Subject: subject}
	if err := identity.Sign(claim, cid, signer, opts...); err != nil {
		return nil, fmt.Errorf("failed to sign claim: %w", err)
	}

	// The certificate is the proof of a spiffe:// subject and is never evaluated for another.
	switch isSPIFFE, hasCertificate := strings.HasPrefix(subject, "spiffe://"), len(claim.GetCertificate()) > 0; {
	case isSPIFFE && !hasCertificate:
		return nil, errors.New("a spiffe:// subject needs a certificate: its X.509-SVID is the proof of the claim")
	case !isSPIFFE && hasCertificate:
		return nil, fmt.Errorf("a certificate is only used for a spiffe:// subject, not %q", subject)
	}

	referrer, err := claim.MarshalReferrer()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claim: %w", err)
	}

	resp, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef:   &corev1.RecordRef{Cid: cid},
		Type:        referrer.GetType(),
		Annotations: referrer.GetAnnotations(),
		CreatedAt:   referrer.GetCreatedAt(),
		Data:        referrer.GetData(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to push claim: %w", err)
	}

	if !resp.GetSuccess() {
		return nil, fmt.Errorf("failed to push claim: %s", resp.GetErrorMessage())
	}

	return claim, nil
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

// Resolve resolves a record name, with an optional version, to CIDs.
// Returns all matching records newest first; if version is empty, all versions.
func (c *Client) Resolve(ctx context.Context, name, version string) (*identityv1.ResolveResponse, error) {
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
