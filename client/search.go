// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
)

// SearchCIDs searches for record CIDs matching the given request.
// Returns a channel that streams CIDs as they are received.
func (c *Client) SearchCIDs(ctx context.Context, req *searchv1.SearchRequest) (<-chan string, error) {
	stream, err := c.SearchServiceClient.SearchCIDs(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search CIDs stream: %w", err)
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
				logger.Error("failed to receive search CIDs response", "error", err)

				return
			}

			select {
			case resultCh <- obj.GetRecordCid():
			case <-ctx.Done():
				logger.Error("context cancelled while receiving search CIDs response", "error", ctx.Err())

				return
			}
		}
	}()

	return resultCh, nil
}

// SearchRecords searches for full records matching the given request.
// Returns a channel that streams records as they are received.
func (c *Client) SearchRecords(ctx context.Context, req *searchv1.SearchRequest) (<-chan *corev1.Record, error) {
	stream, err := c.SearchServiceClient.SearchRecords(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create search records stream: %w", err)
	}

	resultCh := make(chan *corev1.Record)

	go func() {
		defer close(resultCh)

		for {
			obj, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				logger.Error("failed to receive search records response", "error", err)

				return
			}

			select {
			case resultCh <- obj.GetRecord():
			case <-ctx.Done():
				logger.Error("context cancelled while receiving search records response", "error", ctx.Err())

				return
			}
		}
	}()

	return resultCh, nil
}
