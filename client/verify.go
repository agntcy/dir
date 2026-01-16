// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"

	signv1 "github.com/agntcy/dir/api/sign/v1"
)

// Verify verifies the signature of the record using server-side verification.
func (c *Client) Verify(ctx context.Context, req *signv1.VerifyRequest) (*signv1.VerifyResponse, error) {
	return c.SignServiceClient.Verify(ctx, req) //nolint:wrapcheck
}
