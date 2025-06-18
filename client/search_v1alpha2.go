// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	searchtypes "github.com/agntcy/dir/api/search/v1alpha2"
)

func (c *Client) SearchV1alpha2(ctx context.Context, req *searchtypes.SearchRequest) (io.Reader, error) {
	stream, err := c.SearchServiceClientV1alpha2.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search stream: %w", err)
	}

	var buffer bytes.Buffer

	for {
		obj, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to receive search response: %w", err)
		}

		if _, err := buffer.WriteString(obj.GetRecordCid()); err != nil {
			return nil, fmt.Errorf("failed to write search response to buffer: %w", err)
		}
	}

	return &buffer, nil
}
