// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	searchv1 "github.com/agntcy/dir/api/search/v1"
)

func (c *Client) Search(ctx context.Context, req *searchv1.SearchRequest) (<-chan string, error) {
	stream, err := c.SearchServiceClient.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search stream: %w", err)
	}

	resultCh := make(chan string)

	go func() {
		defer close(resultCh)

		for {
			obj, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				logger.Error("failed to receive search response", "error", err)

				return
			}

			select {
			case resultCh <- obj.GetRecordCid():
			case <-ctx.Done():
				logger.Error("context cancelled while receiving search response", "error", ctx.Err())

				return
			}
		}
	}()

	return resultCh, nil
}
