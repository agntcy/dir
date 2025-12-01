// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
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

func (c *searchCtlr) SearchCIDs(req *searchv1.SearchRequest, srv searchv1.SearchService_SearchCIDsServer) error {
	searchLogger.Debug("Called search controller's SearchCIDs method", "req", req)

	filterOptions, err := databaseutils.QueryToFilters(req.GetQueries())
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	filterOptions = append(filterOptions,
		types.WithLimit(int(req.GetLimit())),
		types.WithOffset(int(req.GetOffset())),
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

func (c *searchCtlr) SearchRecords(req *searchv1.SearchRequest, srv searchv1.SearchService_SearchRecordsServer) error {
	searchLogger.Debug("Called search controller's SearchRecords method", "req", req)

	filterOptions, err := databaseutils.QueryToFilters(req.GetQueries())
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	filterOptions = append(filterOptions,
		types.WithLimit(int(req.GetLimit())),
		types.WithOffset(int(req.GetOffset())),
	)

	// Get CIDs from search index
	recordCIDs, err := c.db.GetRecordCIDs(filterOptions...)
	if err != nil {
		return fmt.Errorf("failed to get record CIDs: %w", err)
	}

	// Pull full records from store and stream them
	for _, cid := range recordCIDs {
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
