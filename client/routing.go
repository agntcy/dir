// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("client")

func (c *Client) Publish(ctx context.Context, ref *corev1.RecordRef) error {
	_, err := c.RoutingServiceClient.Publish(ctx, &routingv1.PublishRequest{
		RecordCid: ref.GetCid(),
	})
	if err != nil {
		return fmt.Errorf("failed to publish object: %w", err)
	}

	return nil
}

func (c *Client) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.LegacyListResponse_Item, error) {
	stream, err := c.RoutingServiceClient.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create list stream: %w", err)
	}

	resCh := make(chan *routingv1.LegacyListResponse_Item, 100) //nolint:mnd

	go func() {
		defer close(resCh)

		for {
			obj, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}

			if err != nil {
				logger.Error("error receiving object", "error", err)

				return
			}

			items := obj.GetLegacyListResponse().GetItems()
			for _, item := range items {
				resCh <- item
			}
		}
	}()

	return resCh, nil
}

func (c *Client) Unpublish(ctx context.Context, ref *corev1.RecordRef) error {
	_, err := c.RoutingServiceClient.Unpublish(ctx, &routingv1.UnpublishRequest{
		RecordCid: ref.GetCid(),
	})
	if err != nil {
		return fmt.Errorf("failed to unpublish object: %w", err)
	}

	return nil
}
