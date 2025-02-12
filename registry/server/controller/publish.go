// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
	"github.com/agntcy/dir/registry/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type publishController struct {
	provider types.Registry
	registrytypes.UnimplementedPublishServiceServer
}

func NewPublishController(provider types.Registry) *publishController {
	return &publishController{
		provider: provider,
		UnimplementedPublishServiceServer: registrytypes.UnimplementedPublishServiceServer{},
	}
}

func (c *publishController) Publish(ctx context.Context, req *registrytypes.PublishRequest) (*emptypb.Empty, error) {
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

	err := c.provider.Publish().Publish(ctx, req.GetRecord().GetName(), digest)
	if err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *publishController) Unpublish(ctx context.Context, req *registrytypes.Record) (*emptypb.Empty, error) {
	if req.GetName() == "" {
		return nil, fmt.Errorf("record name is required")
	}

	err := c.provider.Publish().Unpublish(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to unpublish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *publishController) Resolve(ctx context.Context, req *registrytypes.Record) (*registrytypes.ObjectRef, error) {
	if req.GetName() == "" {
		return nil, fmt.Errorf("record name is required")
	}

	digest, err := c.provider.Publish().Resolve(ctx, req.GetName())
	if err != nil {
		return nil, fmt.Errorf("failed to resolve: %w", err)
	}

	return &registrytypes.ObjectRef{
		Digest: &coretypes.Digest{
			Type:  digest.Type,
			Value: digest.Value,
		},
	}, nil
}
