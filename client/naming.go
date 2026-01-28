// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	namingv1 "github.com/agntcy/dir/api/naming/v1"
)

// GetVerificationInfo retrieves the verification info for a record by CID.
func (c *Client) GetVerificationInfo(ctx context.Context, cid string) (*namingv1.GetVerificationInfoResponse, error) {
	if cid == "" {
		return nil, errors.New("cid is required")
	}

	resp, err := c.NamingServiceClient.GetVerificationInfo(ctx, &namingv1.GetVerificationInfoRequest{
		Cid: &cid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get verification info: %w", err)
	}

	return resp, nil
}

// GetVerificationInfoByName retrieves the verification info for a record by name.
// If version is empty, the latest version is used.
func (c *Client) GetVerificationInfoByName(ctx context.Context, name string, version string) (*namingv1.GetVerificationInfoResponse, error) {
	if name == "" {
		return nil, errors.New("name is required")
	}

	req := &namingv1.GetVerificationInfoRequest{
		Name: &name,
	}

	if version != "" {
		req.Version = &version
	}

	resp, err := c.NamingServiceClient.GetVerificationInfo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get verification info: %w", err)
	}

	return resp, nil
}

// Resolve resolves a record reference (name with optional version) to CIDs.
// Returns all matching records sorted by created_at descending (newest first).
// If version is empty, returns all versions; otherwise returns matches for the specific version.
func (c *Client) Resolve(ctx context.Context, name string, version string) (*namingv1.ResolveResponse, error) {
	req := &namingv1.ResolveRequest{
		Name: name,
	}

	if version != "" {
		req.Version = &version
	}

	resp, err := c.NamingServiceClient.Resolve(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve record: %w", err)
	}

	return resp, nil
}
