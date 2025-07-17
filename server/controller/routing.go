// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/opencontainers/go-digest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to get agent: %s", st.Message())
	}

	err = c.routing.Publish(ctx, &coretypes.Object{
		Ref:   ref,
		Agent: agent.Agent,
	})
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to publish: %s", st.Message())
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) List(req *routingtypes.ListRequest, srv routingtypes.RoutingService_ListServer) error {
	routingLogger.Debug("Called routing controller's List method", "req", req)

	itemChan, err := c.routing.List(srv.Context(), req)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to list: %s", st.Message())
	}

	items := []*routingtypes.ListResponse_Item{}
	for i := range itemChan {
		items = append(items, i)
	}

	if err := srv.Send(&routingtypes.ListResponse{Items: items}); err != nil {
		return status.Errorf(codes.Internal, "failed to send list response: %v", err)
	}

	return nil
}

func (c *routingCtlr) Unpublish(ctx context.Context, req *routingtypes.UnpublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Unpublish method", "req", req)

	ref, agent, err := c.getAgent(ctx, req.GetRecord())
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to get agent: %s", st.Message())
	}

	err = c.routing.Unpublish(ctx, &coretypes.Object{
		Ref:   ref,
		Agent: agent.Agent,
	})
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to unpublish: %s", st.Message())
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) getAgent(ctx context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, *coretypes.Agent, error) {
	routingLogger.Debug("Called routing controller's getAgent method", "ref", ref)

	if ref == nil || ref.GetType() == "" {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object reference is required and must have a type")
	}

	if ref.GetDigest() == "" {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object reference must have a digest")
	}

	_, err := digest.Parse(ref.GetDigest())
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid digest: %v", err)
	}

	ref, err = c.store.Lookup(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, nil, status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	if ref.GetSize() > 4*1024*1024 {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object size exceeds maximum limit of 4MB")
	}

	if ref.GetType() != coretypes.ObjectType_OBJECT_TYPE_AGENT.String() {
		return nil, nil, status.Errorf(codes.InvalidArgument, "object type must be %s", coretypes.ObjectType_OBJECT_TYPE_AGENT.String())
	}

	reader, err := c.store.Pull(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, nil, status.Errorf(st.Code(), "failed to pull object: %s", st.Message())
	}

	agent := &coretypes.Agent{}

	_, err = agent.LoadFromReader(reader)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to load agent from reader: %v", err)
	}

	return ref, agent, nil
}
