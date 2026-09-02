// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var searchLogger = logging.Logger("controller/search")

type searchCtlr struct {
	searchv1.UnimplementedSearchServiceServer
	db    types.DatabaseAPI
	store types.StoreAPI
}

func NewSearchController(db types.DatabaseAPI, store types.StoreAPI) searchv1.SearchServiceServer {
	return &searchCtlr{
		UnimplementedSearchServiceServer: searchv1.UnimplementedSearchServiceServer{},
		db:                               db,
		store:                            store,
	}
}

func (c *searchCtlr) CountRecords(_ context.Context, req *searchv1.CountRecordsRequest) (*searchv1.CountRecordsResponse, error) {
	searchLogger.Debug("Called search controller's CountRecords method", "req", req)

	filterOptions, err := databaseutils.QueryToFilters(req.GetQueries())
	if err != nil {
		return nil, fmt.Errorf("failed to create filter options: %w", err)
	}

	totalCount, err := c.db.CountRecords(filterOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	return &searchv1.CountRecordsResponse{TotalCount: totalCount}, nil
}

func (c *searchCtlr) ListRecordValues(_ context.Context, req *searchv1.ListRecordValuesRequest) (*searchv1.ListRecordValuesResponse, error) {
	searchLogger.Debug("Called search controller's ListRecordValues method", "req", req)

	// Reject unsupported fields up front so a caller gets a precise error rather
	// than a response that silently omits what it asked for.
	for _, field := range req.GetFields() {
		if !types.IsSupportedRecordValueField(field) {
			return nil, status.Errorf(codes.InvalidArgument,
				"unsupported field %s: supported fields are %v", field, types.SupportedRecordValueFields())
		}
	}

	fieldValues, err := c.db.ListRecordValues(req.GetFields())
	if err != nil {
		return nil, fmt.Errorf("failed to list record values: %w", err)
	}

	fields := make([]*searchv1.ListRecordValuesResponse_FieldValues, 0, len(fieldValues))
	for _, fieldValue := range fieldValues {
		fields = append(fields, &searchv1.ListRecordValuesResponse_FieldValues{
			Field:  fieldValue.Field,
			Values: fieldValue.Values,
		})
	}

	return &searchv1.ListRecordValuesResponse{Fields: fields}, nil
}

func (c *searchCtlr) SearchCIDs(req *searchv1.SearchCIDsRequest, srv searchv1.SearchService_SearchCIDsServer) error {
	searchLogger.Debug("Called search controller's SearchCIDs method", "req", req)

	filterOptions, err := databaseutils.QueryToFilters(req.GetQueries())
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	filterOptions = append(filterOptions,
		types.WithLimit(int(req.GetLimit())),
		types.WithOffset(int(req.GetOffset())),
		sortModeToOrderBy(req.GetSortMode()),
	)

	recordCIDs, err := c.db.GetRecordCIDs(filterOptions...)
	if err != nil {
		return fmt.Errorf("failed to get record CIDs: %w", err)
	}

	for _, cid := range recordCIDs {
		if err := srv.Send(&searchv1.SearchCIDsResponse{RecordCid: cid}); err != nil {
			return fmt.Errorf("failed to send record CID: %w", err)
		}
	}

	return nil
}

func (c *searchCtlr) SearchRecords(req *searchv1.SearchRecordsRequest, srv searchv1.SearchService_SearchRecordsServer) error {
	searchLogger.Debug("Called search controller's SearchRecords method", "req", req)

	filterOptions, err := databaseutils.QueryToFilters(req.GetQueries())
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	filterOptions = append(filterOptions,
		types.WithLimit(int(req.GetLimit())),
		types.WithOffset(int(req.GetOffset())),
		sortModeToOrderBy(req.GetSortMode()),
	)

	recordCIDs, err := c.db.GetRecordCIDs(filterOptions...)
	if err != nil {
		return fmt.Errorf("failed to get record CIDs: %w", err)
	}

	for _, cid := range recordCIDs {
		if err := srv.Context().Err(); err != nil {
			return fmt.Errorf("client disconnected: %w", err)
		}

		record, err := c.store.Pull(srv.Context(), &corev1.RecordRef{Cid: cid})
		if err != nil {
			searchLogger.Warn("Failed to pull record from store", "cid", cid, "error", err)

			continue
		}

		if err := srv.Send(&searchv1.SearchRecordsResponse{Record: record}); err != nil {
			return fmt.Errorf("failed to send record: %w", err)
		}
	}

	return nil
}

// sortModeToOrderBy converts a proto SortMode into a WithOrderBy filter option.
func sortModeToOrderBy(mode searchv1.SortMode) types.FilterOption {
	switch mode {
	case searchv1.SortMode_SORT_MODE_POPULARITY:
		return types.WithOrderBy(types.RecordOrderClause{Column: "popularity_score", Desc: true})
	case searchv1.SortMode_SORT_MODE_PROVIDER_COUNT:
		return types.WithOrderBy(types.RecordOrderClause{Column: "provider_count", Desc: true})
	case searchv1.SortMode_SORT_MODE_RELEVANCE:
		return types.WithOrderBy()
	case searchv1.SortMode_SORT_MODE_UNSPECIFIED, searchv1.SortMode_SORT_MODE_RECENCY:
		return types.WithOrderBy()
	}

	return types.WithOrderBy()
}
