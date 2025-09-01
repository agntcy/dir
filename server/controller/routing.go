// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

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
	routing     types.RoutingAPI
	store       types.StoreAPI
	publication types.PublicationAPI
}

func NewRoutingController(routing types.RoutingAPI, store types.StoreAPI, publication types.PublicationAPI) routingv1.RoutingServiceServer {
	return &routingCtlr{
		routing:                           routing,
		store:                             store,
		publication:                       publication,
		UnimplementedRoutingServiceServer: routingv1.UnimplementedRoutingServiceServer{},
	}
}

func (c *routingCtlr) Publish(ctx context.Context, req *routingv1.PublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Publish method", "req", req)

	// Create publication to be handled by the publication service
	publicationID, err := c.publication.CreatePublication(ctx, req)
	if err != nil {
		routingLogger.Error("Failed to create publication", "error", err)

		return nil, status.Errorf(codes.Internal, "failed to create publication: %v", err)
	}

	routingLogger.Info("Publication created successfully", "publication_id", publicationID)

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

func (c *routingCtlr) Unpublish(_ context.Context, req *routingv1.UnpublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Unpublish method", "req", req)

	// Unpublish is intentionally not implemented.
	// Records will be deleted from the network once their retention period (TTL) expires.

	return &emptypb.Empty{}, nil
}
