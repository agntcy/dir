// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type publishController struct {
	publish types.PublishService
	routingtypes.UnimplementedPublishServiceServer
}

func NewPublishController(publish types.PublishService) routingtypes.PublishServiceServer {
	return &publishController{
		publish:                           publish,
		UnimplementedPublishServiceServer: routingtypes.UnimplementedPublishServiceServer{},
	}
}

func (c *publishController) Publish(ctx context.Context, req *routingtypes.PublishRequest) (*emptypb.Empty, error) {
	if req.GetRecord() == nil || req.GetRecord().GetName() == "" {
		return nil, fmt.Errorf("record name is required")
	}
	if req.GetRef() == nil || req.GetRef().GetDigest() == nil || len(req.GetRef().GetDigest().GetValue()) == 0 {
		return nil, fmt.Errorf("digest is required")
	}

	digest := &coretypes.Digest{
		Type:  req.GetRef().GetDigest().GetType(),
		Value: req.GetRef().GetDigest().GetValue(),
	}

	err := c.publish.Publish(ctx, req.GetRecord().GetName(), digest)
	if err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *publishController) Unpublish(ctx context.Context, req *routingtypes.Record) (*emptypb.Empty, error) {
	if req.GetName() == "" {
		return nil, fmt.Errorf("record name is required")
	}

	err := c.publish.Unpublish(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to unpublish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *publishController) Resolve(ctx context.Context, req *routingtypes.Record) (*coretypes.ObjectRef, error) {
	if req.GetName() == "" {
		return nil, fmt.Errorf("record name is required")
	}

	digest, err := c.publish.Resolve(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve: %w", err)
	}

	return &coretypes.ObjectRef{
		Digest: digest,
	}, nil
}
