// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"context"

	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	"github.com/agntcy/dir/server/controller"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

var routingLogger = logging.Logger("controller/v1alpha2/routing")

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
	// TODO: implement v1alpha2 routing publish logic with converter
	routingLogger.Debug("Called v1alpha2 routing controller's Publish method", "req", req)
	return nil, status.Errorf(codes.Unimplemented, "method Publish not implemented")
}

func (c *routingCtlr) Unpublish(ctx context.Context, req *routingtypes.UnpublishRequest) (*emptypb.Empty, error) {
	// TODO: implement v1alpha2 routing unpublish logic with converter
	routingLogger.Debug("Called v1alpha2 routing controller's Unpublish method", "req", req)
	return nil, status.Errorf(codes.Unimplemented, "method Unpublish not implemented")
}

func (c *routingCtlr) Search(req *routingtypes.SearchRequest, srv routingtypes.RoutingService_SearchServer) error {
	// TODO: implement v1alpha2 routing search logic with converter
	routingLogger.Debug("Called v1alpha2 routing controller's Search method", "req", req)
	return status.Errorf(codes.Unimplemented, "method Search not implemented")
}

func (c *routingCtlr) List(req *routingtypes.ListRequest, srv routingtypes.RoutingService_ListServer) error {
	// TODO: implement v1alpha2 routing list logic with converter
	routingLogger.Debug("Called v1alpha2 routing controller's List method", "req", req)
	return status.Errorf(codes.Unimplemented, "method List not implemented")
}
