// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var publicationLogger = logging.Logger("controller/publication")

// publicationCtlr implements the PublicationService gRPC interface.
type publicationCtlr struct {
	routingv1.UnimplementedPublicationServiceServer
	db   types.DatabaseAPI
	opts types.APIOptions
}

// NewPublicationController creates a new publication controller.
func NewPublicationController(db types.DatabaseAPI, opts types.APIOptions) routingv1.PublicationServiceServer {
	return &publicationCtlr{
		db:   db,
		opts: opts,
	}
}

func (c *publicationCtlr) CreatePublication(_ context.Context, req *routingv1.PublishRequest) (*routingv1.CreatePublicationResponse, error) {
	publicationLogger.Debug("Called publication controller's CreatePublication method")

	// Validate the publish request
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "publish request cannot be nil")
	}

	// Validate that at least one request type is specified
	switch req.GetRequest().(type) {
	case *routingv1.PublishRequest_RecordRefs:
		if req.GetRecordRefs() == nil || len(req.GetRecordRefs().GetRefs()) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "record refs cannot be empty")
		}
	case *routingv1.PublishRequest_Queries:
		if req.GetQueries() == nil || len(req.GetQueries().GetQueries()) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "queries cannot be empty")
		}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid publish request: must specify record_refs, queries, or all_records")
	}

	id, err := c.db.CreatePublication(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create publication: %w", err)
	}

	publicationLogger.Debug("Publication created successfully", "publication_id", id)

	return &routingv1.CreatePublicationResponse{
		PublicationId: id,
	}, nil
}

func (c *publicationCtlr) ListPublications(req *routingv1.ListPublicationsRequest, srv routingv1.PublicationService_ListPublicationsServer) error {
	publicationLogger.Debug("Called publication controller's ListPublications method", "req", req)

	offset := int(req.GetOffset())
	limit := int(req.GetLimit())

	publications, err := c.db.GetPublications(offset, limit)
	if err != nil {
		return fmt.Errorf("failed to list publications: %w", err)
	}

	for _, publication := range publications {
		publicationLogger.Debug("Sending publication object", "publication_id", publication.GetID(), "status", publication.GetStatus())

		if err := srv.Send(&routingv1.ListPublicationsItem{
			PublicationId:  publication.GetID(),
			Status:         publication.GetStatus(),
			CreatedTime:    publication.GetCreatedTime(),
			LastUpdateTime: publication.GetLastUpdateTime(),
		}); err != nil {
			return fmt.Errorf("failed to send publication object: %w", err)
		}
	}

	publicationLogger.Debug("Finished sending publication objects")

	return nil
}

func (c *publicationCtlr) GetPublication(_ context.Context, req *routingv1.GetPublicationRequest) (*routingv1.GetPublicationResponse, error) {
	publicationLogger.Debug("Called publication controller's GetPublication method", "req", req)

	if req.GetPublicationId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "publication_id cannot be empty")
	}

	publicationObj, err := c.db.GetPublicationByID(req.GetPublicationId())
	if err != nil {
		return nil, fmt.Errorf("failed to get publication by ID: %w", err)
	}

	return &routingv1.GetPublicationResponse{
		PublicationId:  publicationObj.GetID(),
		Status:         publicationObj.GetStatus(),
		CreatedTime:    publicationObj.GetCreatedTime(),
		LastUpdateTime: publicationObj.GetLastUpdateTime(),
	}, nil
}
