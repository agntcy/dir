// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var aiFinderLogger = logging.Logger("controller/ai-finder")

// aiFinderController adapts the AI Finder query language to the catalog query
// layer. GetWellKnownCatalog is served by the embedded Unimplemented server.
type aiFinderController struct {
	catalogv1.UnimplementedAIFinderServiceServer

	db types.CatalogDatabaseAPI
}

func NewAIFinderController(db types.CatalogDatabaseAPI) catalogv1.AIFinderServiceServer {
	return &aiFinderController{db: db}
}

// ListAgents parses the filter, order, and paging arguments, queries the
// catalog, and returns one page of entries with a continuation token.
func (c *aiFinderController) ListAgents(ctx context.Context, req *catalogv1.ListAgentsRequest) (*catalogv1.ListAgentsResponse, error) {
	if req == nil {
		req = &catalogv1.ListAgentsRequest{}
	}

	aiFinderLogger.Debug("ListAgents called", "filter", req.GetFilter(), "order_by", req.GetOrderBy(), "page_size", req.GetPageSize())

	parsedFilter, err := parseAgentFilter(req.GetFilter())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
	}

	// publisherId is part of the grammar but not yet indexed.
	if len(parsedFilter.PublisherIDs) > 0 {
		return nil, status.Error(codes.Unimplemented, "publisherId filter is not yet supported") //nolint:wrapcheck
	}

	order, err := parseOrderBy(req.GetOrderBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_by: %v", err)
	}

	offset, err := decodePageToken(req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
	}

	pageSize := int(clampPageSize(req.GetPageSize()))

	opts, ok := buildRecordFilterOptions(parsedFilter, order, pageSize, offset)
	if !ok {
		// type= matched no indexed module: zero rows, not an error.
		return &catalogv1.ListAgentsResponse{}, nil
	}

	entries, hasMore, err := c.db.GetCatalogEntries(opts...)
	if err != nil {
		aiFinderLogger.Error("failed to list catalog entries", "error", err)

		return nil, status.Error(codes.Internal, "failed to list catalog entries") //nolint:wrapcheck
	}

	var nextPageToken string
	if hasMore {
		// Advance by the page size: the number of records consumed,
		// regardless of how many projected to an entry.
		nextPageToken = encodePageToken(offset + pageSize)
	}

	return &catalogv1.ListAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
	}, nil
}
