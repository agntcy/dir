// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/opencontainers/go-digest"
	"google.golang.org/protobuf/types/known/emptypb"
)

var routingLogger = logging.Logger("controller/routing")

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
	routingLogger.Debug("Called routing controller's Publish method", "req", req)

	ref, agent, err := c.getAgent(ctx, req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	err = c.routing.Publish(ctx, &coretypes.Object{
		Ref:   ref,
		Agent: agent,
	}, req.GetNetwork())
	if err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) List(req *routingtypes.ListRequest, srv routingtypes.RoutingService_ListServer) error {
	routingLogger.Debug("Called routing controller's List method", "req", req)

	itemChan, err := c.routing.List(srv.Context(), req)
	if err != nil {
		return fmt.Errorf("failed to list: %w", err)
	}

	items := []*routingtypes.ListResponse_Item{}
	for i := range itemChan {
		items = append(items, i)
	}

	if err := srv.Send(&routingtypes.ListResponse{Items: items}); err != nil {
		return fmt.Errorf("failed to send: %w", err)
	}

	return nil
}

func (c *routingCtlr) Unpublish(ctx context.Context, req *routingtypes.UnpublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Unpublish method", "req", req)

	ref, agent, err := c.getAgent(ctx, req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	err = c.routing.Unpublish(ctx, &coretypes.Object{
		Ref:   ref,
		Agent: agent,
	}, req.GetNetwork())
	if err != nil {
		return nil, fmt.Errorf("failed to publish: %w", err)
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) getAgent(ctx context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, *coretypes.Agent, error) {
	routingLogger.Debug("Called routing controller's getAgent method", "ref", ref)

	if ref == nil || ref.GetType() == "" {
		return nil, nil, errors.New("record is required")
	}

	if ref.GetDigest() == "" {
		return nil, nil, errors.New("digest is required")
	}

	_, err := digest.Parse(ref.GetDigest())
	if err != nil {
		return nil, nil, fmt.Errorf("invalid digest: %w", err)
	}

	ref, err = c.store.Lookup(ctx, ref)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to lookup: %w", err)
	}

	if ref.GetSize() > 4*1024*1024 {
		return nil, nil, errors.New("size is too large")
	}

	if ref.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return nil, nil, errors.New("unsupported object type")
	}

	reader, err := c.store.Pull(ctx, ref)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to pull: %w", err)
	}

	agent := &coretypes.Agent{}

	_, err = agent.LoadFromReader(reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load agent: %w", err)
	}

	return ref, agent, nil
}
