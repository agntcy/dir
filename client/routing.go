// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
)

func (c *Client) Publish(ctx context.Context, ref *coretypes.ObjectRef, local bool) error {
	_, err := c.RoutingServiceClient.Publish(ctx, &routingtypes.PublishRequest{
		Record: ref,
		Local:  &local,
	})
	if err != nil {
		return fmt.Errorf("failed to publish object: %w", err)
	}

	return nil
}

func (c *Client) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	stream, err := c.RoutingServiceClient.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	resCh := make(chan *routingtypes.ListResponse_Item, 100)

	go func() {
		defer close(resCh)

		for {
			items, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				log.Printf("failed to receive object: %v", err)
				continue
			}

			for _, item := range items.GetItems() {
				resCh <- item
			}
		}
	}()

	return resCh, nil
}
