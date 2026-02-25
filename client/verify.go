// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"
	"strings"

	signv1 "github.com/agntcy/dir/api/sign/v1"
)

// Verify returns the cached signature verification result for a record by querying the server.
// Verification is performed by the reconciler; this call only reads from the database.
func (c *Client) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	if req.GetRecordRef() == nil || req.GetRecordRef().GetCid() == "" {
		return nil, fmt.Errorf("record CID is required")
	}

	_, err := c.Lookup(ctx, req.GetRecordRef())
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			errMsg := "record not found"

			return &signv1.VerifyResponse{
				Success:      false,
				ErrorMessage: &errMsg,
			}, nil
		}

		return nil, fmt.Errorf("failed to lookup record: %w", err)
	}

	resp, err := c.SignServiceClient.Verify(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}

	return resp, nil
}
