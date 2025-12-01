// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/streaming"
)

// SearchCIDs searches for record CIDs matching the given request.
func (c *Client) SearchCIDs(ctx context.Context, req *searchv1.SearchRequest) (streaming.StreamResult[searchv1.SearchCIDsResponse], error) {
	stream, err := c.SearchServiceClient.SearchCIDs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search CIDs stream: %w", err)
	}

	return streaming.ProcessServerStream(ctx, stream)
}

// SearchRecords searches for full records matching the given request.
func (c *Client) SearchRecords(ctx context.Context, req *searchv1.SearchRequest) (streaming.StreamResult[searchv1.SearchRecordsResponse], error) {
	stream, err := c.SearchServiceClient.SearchRecords(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search records stream: %w", err)
	}

	return streaming.ProcessServerStream(ctx, stream)
}
