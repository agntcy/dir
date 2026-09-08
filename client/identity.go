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
)

// ClaimIdentity signs and pushes an identity claim for recordCID, asserting
// that subject is the record's own claimed identity.
func (c *Client) ClaimIdentity(ctx context.Context, recordCID, subject string, signer identityv1.Signer) error {
	claim := identityv1.NewIdentityClaim(subject)
	if err := identityv1.SignIdentityClaim(claim, recordCID, signer); err != nil {
		return fmt.Errorf("failed to sign identity claim: %w", err)
	}

	ref, err := claim.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("failed to marshal identity claim referrer: %w", err)
	}

	return c.pushClaimReferrer(ctx, recordCID, ref)
}

// ClaimOwnership signs and pushes an ownership claim for recordCID, asserting
// that subject owns/controls the record.
func (c *Client) ClaimOwnership(ctx context.Context, recordCID, subject string, signer identityv1.Signer) error {
	claim := identityv1.NewOwnershipClaim(subject)
	if err := identityv1.SignOwnershipClaim(claim, recordCID, signer); err != nil {
		return fmt.Errorf("failed to sign ownership claim: %w", err)
	}

	ref, err := claim.MarshalReferrer()
	if err != nil {
		return fmt.Errorf("failed to marshal ownership claim referrer: %w", err)
	}

	return c.pushClaimReferrer(ctx, recordCID, ref)
}

func (c *Client) pushClaimReferrer(ctx context.Context, recordCID string, ref *corev1.RecordReferrer) error {
	resp, err := c.PushReferrer(ctx, &storev1.PushReferrerRequest{
		RecordRef: &corev1.RecordRef{Cid: recordCID},
		Type:      ref.GetType(),
		Data:      ref.GetData(),
	})
	if err != nil {
		return fmt.Errorf("failed to push claim referrer: %w", err)
	}

	if !resp.GetSuccess() {
		return fmt.Errorf("push claim referrer failed: %s", resp.GetErrorMessage())
	}

	return nil
}

// GetIdentityStatus retrieves the cached identity/ownership claim
// verification status for a record by CID.
func (c *Client) GetIdentityStatus(ctx context.Context, cid string) (*identityv1.GetIdentityStatusResponse, error) {
	if cid == "" {
		return nil, errors.New("cid is required")
	}

	resp, err := c.identityClient.GetIdentityStatus(ctx, &identityv1.GetIdentityStatusRequest{Cid: &cid})
	if err != nil {
		return nil, fmt.Errorf("failed to get identity status: %w", err)
	}

	return resp, nil
}

// GetIdentityStatusByName retrieves the cached identity/ownership claim
// verification status for a record by name. If version is empty, the latest
// version is used.
func (c *Client) GetIdentityStatusByName(ctx context.Context, name, version string) (*identityv1.GetIdentityStatusResponse, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	req := &identityv1.GetIdentityStatusRequest{Name: &name}
	if version != "" {
		req.Version = &version
	}

	resp, err := c.identityClient.GetIdentityStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get identity status: %w", err)
	}

	return resp, nil
}

// ResolveIdentity resolves a record reference (name with optional version) to
// CIDs via IdentityService.Resolve. If version is empty, returns all
// versions (newest first). Named distinctly from the naming.v1-based
// Client.Resolve (client/naming.go) so both can coexist.
func (c *Client) ResolveIdentity(ctx context.Context, name, version string) (*identityv1.ResolveResponse, error) {
	req := &identityv1.ResolveRequest{Name: name}
	if version != "" {
		req.Version = &version
	}

	resp, err := c.identityClient.Resolve(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve record: %w", err)
	}

	return resp, nil
}
