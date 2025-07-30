// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var routingLogger = logging.Logger("controller/routing")

type routingCtlr struct {
	routingv1.UnimplementedRoutingServiceServer
	routing types.RoutingAPI
	store   types.StoreAPI
}

func NewRoutingController(routing types.RoutingAPI, store types.StoreAPI) routingv1.RoutingServiceServer {
	return &routingCtlr{
		routing:                           routing,
		store:                             store,
		UnimplementedRoutingServiceServer: routingv1.UnimplementedRoutingServiceServer{},
	}
}

func (c *routingCtlr) Publish(ctx context.Context, req *routingv1.PublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Publish method", "req", req)

	ref := &corev1.RecordRef{
		Cid: req.GetRecordCid(),
	}

	record, err := c.getRecord(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to get record: %s", st.Message())
	}

	err = c.routing.Publish(ctx, ref, record)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to publish: %s", st.Message())
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) List(req *routingv1.ListRequest, srv routingv1.RoutingService_ListServer) error {
	routingLogger.Debug("Called routing controller's List method", "req", req)

	itemChan, err := c.routing.List(srv.Context(), req)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to list: %s", st.Message())
	}

	items := []*routingv1.LegacyListResponse_Item{}
	for i := range itemChan {
		items = append(items, i)
	}

	if err := srv.Send(&routingv1.ListResponse{
		LegacyListResponse: &routingv1.LegacyListResponse{
			Items: items,
		},
	}); err != nil {
		return status.Errorf(codes.Internal, "failed to send list response: %v", err)
	}

	return nil
}

func (c *routingCtlr) Unpublish(ctx context.Context, req *routingv1.UnpublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Unpublish method", "req", req)

	ref := &corev1.RecordRef{
		Cid: req.GetRecordCid(),
	}

	record, err := c.getRecord(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to get record: %s", st.Message())
	}

	err = c.routing.Unpublish(ctx, ref, record)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to unpublish: %s", st.Message())
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) getRecord(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	routingLogger.Debug("Called routing controller's getRecord method", "ref", ref)

	if ref == nil || ref.GetCid() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "object reference is required and must have a type")
	}

	_, err := c.store.Lookup(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to lookup object: %s", st.Message())
	}

	record, err := c.store.Pull(ctx, ref)
	if err != nil {
		st := status.Convert(err)

		return nil, status.Errorf(st.Code(), "failed to pull object: %s", st.Message())
	}

	return record, nil
}
