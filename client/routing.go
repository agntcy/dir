// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
)

func (c *Client) Publish(ctx context.Context, ref *coretypes.ObjectRef) error {
	_, err := c.RoutingServiceClient.Publish(ctx, &routingtypes.PublishRequest{
		Record: ref,
		Peer:   nil, // TODO check if required in client
	})
	if err != nil {
		return fmt.Errorf("failed to publish object: %w", err)
	}

	return nil
}

func (c *Client) List(ctx context.Context, query string) ([]*coretypes.ObjectRef, error) {
	stream, err := c.RoutingServiceClient.List(ctx, &routingtypes.ListRequest{
		Query: query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pull stream: %w", err)
	}

	var refs []*coretypes.ObjectRef

	for {
		obj, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to receive object: %w", err)
		}

		items := obj.GetItems()
		for _, item := range items {
			// TODO check if item peer and key are required in client
			refs = append(refs, item.GetRecord())
		}
	}

	// TODO return read channel
	return refs, nil
}
