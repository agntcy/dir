// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"context"

	"github.com/agntcy/dir/api/converter"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var routingLogger = logging.Logger("controller/v1alpha1/routing")

type routingCtlr struct {
	routingtypes.UnimplementedRoutingServiceServer
	commonController *controller.RoutingController
}

func NewRoutingController(routing types.RoutingAPI, store types.StoreAPI) routingtypes.RoutingServiceServer {
	return &routingCtlr{
		UnimplementedRoutingServiceServer: routingtypes.UnimplementedRoutingServiceServer{},
		commonController:                  controller.NewRoutingController(routing, store),
	}
}

func (c *routingCtlr) Publish(ctx context.Context, req *routingtypes.PublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called v1alpha1 routing controller's Publish method", "req", req)

	// Convert core.v1alpha1.ObjectRef to objectmanager.RecordObject
	recordObject, err := converter.CoreV1Alpha1ToRecordObject(&coretypes.Object{Ref: req.GetRecord()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert object ref: %v", err)
	}

	// Use common controller to handle business logic
	err = c.commonController.Publish(ctx, recordObject, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) List(req *routingtypes.ListRequest, srv routingtypes.RoutingService_ListServer) error {
	routingLogger.Debug("Called v1alpha1 routing controller's List method", "req", req)

	// Use common controller to handle business logic
	itemChan, err := c.commonController.List(srv.Context(), req)
	if err != nil {
		return err
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
	routingLogger.Debug("Called v1alpha1 routing controller's Unpublish method", "req", req)

	// Convert core.v1alpha1.ObjectRef to objectmanager.RecordObject
	recordObject, err := converter.CoreV1Alpha1ToRecordObject(&coretypes.Object{Ref: req.GetRecord()})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert object ref: %v", err)
	}

	// Use common controller to handle business logic
	err = c.commonController.Unpublish(ctx, recordObject, req.GetNetwork())
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}
