// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
)

func (c *Client) Publish(ctx context.Context, ref *coretypes.ObjectRef) error {
	_, err := c.RoutingServiceClient.Publish(ctx, &routingtypes.PublishRequest{
		Record: ref,
	})
	if err != nil {
		return fmt.Errorf("failed to publish object: %w", err)
	}

	return nil
}

func (c *Client) List(ctx context.Context, req *routingtypes.ListRequest) (<-chan *routingtypes.ListResponse_Item, error) {
	_, err := c.RoutingServiceClient.List(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	// var refs []*coretypes.ObjectRef

	// for {
	// 	obj, err := stream.Recv()
	// 	if errors.Is(err, io.EOF) {
	// 		break
	// 	}

	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to receive object: %w", err)
	// 	}

	// 	items := obj.GetItems()
	// 	for _, item := range items {
	// 		// TODO check if item peer and key are required in client
	// 		refs = append(refs, item.GetRecord())
	// 	}
	// }

	// TODO return read channel
	return nil, errors.New("not implemented")
}
