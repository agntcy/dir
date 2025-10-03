// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
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

	// Stream ListResponse items directly to the client
	for item := range itemChan {
		if err := srv.Send(item); err != nil {
			return status.Errorf(codes.Internal, "failed to send list response: %v", err)
		}
	}

	return nil
}

func (c *routingCtlr) Search(req *routingv1.SearchRequest, srv routingv1.RoutingService_SearchServer) error {
	routingLogger.Debug("Called routing controller's Search method", "req", req)

	itemChan, err := c.routing.Search(srv.Context(), req)
	if err != nil {
		st := status.Convert(err)

		return status.Errorf(st.Code(), "failed to search: %s", st.Message())
	}

	// Stream SearchResponse items directly to the client
	for item := range itemChan {
		if err := srv.Send(item); err != nil {
			return status.Errorf(codes.Internal, "failed to send search response: %v", err)
		}
	}

	return nil
}

func (c *routingCtlr) Unpublish(ctx context.Context, req *routingv1.UnpublishRequest) (*emptypb.Empty, error) {
	routingLogger.Debug("Called routing controller's Unpublish method", "req", req)

	// Only handle RecordRefs, not queries
	recordRefs, ok := req.GetRequest().(*routingv1.UnpublishRequest_RecordRefs)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "unpublish request must specify record_refs") //nolint:wrapcheck // gRPC status errors should not be wrapped
	}

	// Process each RecordRef
	for _, ref := range recordRefs.RecordRefs.GetRefs() {
		record, err := c.getRecord(ctx, ref)
		if err != nil {
			st := status.Convert(err)

			return nil, status.Errorf(st.Code(), "failed to get record: %s", st.Message())
		}

		// Wrap record with adapter for interface-based unpublishing
		adapter := adapters.NewRecordAdapter(record)

		err = c.routing.Unpublish(ctx, adapter)
		if err != nil {
			st := status.Convert(err)

			return nil, status.Errorf(st.Code(), "failed to unpublish: %s", st.Message())
		}

		routingLogger.Info("Successfully unpublished record", "cid", ref.GetCid())
	}

	return &emptypb.Empty{}, nil
}

func (c *routingCtlr) getRecord(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	routingLogger.Debug("Called routing controller's getRecord method", "ref", ref)

	if ref == nil || ref.GetCid() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "object reference is required and must have a CID")
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
