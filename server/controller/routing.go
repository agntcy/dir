// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type routingCtlr struct {
	routingtypes.UnimplementedRoutingServiceServer
	routing types.RoutingAPI
}

func NewRoutingController(routing types.RoutingAPI) routingtypes.RoutingServiceServer {
	return &routingCtlr{
		routing:                           routing,
		UnimplementedRoutingServiceServer: routingtypes.UnimplementedRoutingServiceServer{},
	}
}

func (c *routingCtlr) Publish(ctx context.Context, req *routingtypes.PublishRequest) (*emptypb.Empty, error) {
	if req.GetRecord() == nil || req.GetRecord().GetType() == "" {
		return nil, errors.New("record is required")
	}

	if req.GetRecord().GetDigest() == "" {
		return nil, errors.New("digest is required")
	}

	err := c.routing.Publish(ctx, req.GetRecord(), &coretypes.Agent{})
	if err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) Lookup(_ context.Context, _ *routingtypes.Key) (*coretypes.ObjectRef, error) {
	// TODO implement me
	panic("implement me")
}

func (c *routingCtlr) Resolve(_ *routingtypes.Key, _ routingtypes.RoutingService_ResolveServer) error {
	// TODO implement me
	panic("implement me")
}

func (c *routingCtlr) List(req *routingtypes.ListRequest, srv routingtypes.RoutingService_ListServer) error {
	refs, err := c.routing.List(context.Background(), req.GetQuery())
	if err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	items := []*routingtypes.ListResponse_Item{}
	for _, ref := range refs {
		items = append(items, &routingtypes.ListResponse_Item{
			Key:    nil, // TODO: required? if yes, get agent from ref and compute key
			Peers:  nil, // TODO: implement me when peers are available
			Record: ref,
		})
	}

	if err := srv.Send(
		&routingtypes.ListResponse{
			Items: items,
		}); err != nil {
		return fmt.Errorf("failed to send ref: %w", err)
	}

	return nil
}
