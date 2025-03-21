// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/opencontainers/go-digest"
	"google.golang.org/protobuf/types/known/emptypb"
)

type routingCtlr struct {
	routingtypes.UnimplementedRoutingServiceServer
	routing types.RoutingAPI
	store   types.StoreAPI
}

func NewRoutingController(routing types.RoutingAPI, store types.StoreAPI) routingtypes.RoutingServiceServer {
	return &routingCtlr{
		routing:                           routing,
		store:                             store,
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

	_, err := digest.Parse(req.GetRecord().GetDigest())
	if err != nil {
		return nil, fmt.Errorf("invalid digest: %w", err)
	}

	ref, err := c.store.Lookup(ctx, req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to lookup: %w", err)
	}

	if ref.GetSize() > 4*1024*1024 {
		return nil, errors.New("size is too large")
	}

	if ref.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return nil, errors.New("unsupported object type")
	}

	reader, err := c.store.Pull(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to pull: %w", err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}

	var agent coretypes.Agent
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent: %w", err)
	}

	err = c.routing.Publish(ctx, req.GetRecord(), &agent)
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
