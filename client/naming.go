// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	namingv1 "github.com/agntcy/dir/api/naming/v1"
)

// GetVerificationInfo retrieves the verification info for a record.
func (c *Client) GetVerificationInfo(ctx context.Context, cid string) (*namingv1.GetVerificationInfoResponse, error) {
	if cid == "" {
		return nil, errors.New("cid is required")
	}

	resp, err := c.NamingServiceClient.GetVerificationInfo(ctx, &namingv1.GetVerificationInfoRequest{
		Cid: cid,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get verification info: %w", err)
	}

	return resp, nil
}
