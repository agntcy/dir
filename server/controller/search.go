// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var searchLogger = logging.Logger("controller/search")

type searchCtlr struct {
	searchv1.UnimplementedSearchServiceServer
	db types.DatabaseAPI
}

func NewSearchController(db types.DatabaseAPI) searchv1.SearchServiceServer {
	return &searchCtlr{
		UnimplementedSearchServiceServer: searchv1.UnimplementedSearchServiceServer{},
		db:                               db,
	}
}

func (c *searchCtlr) Search(req *searchv1.SearchRequest, srv searchv1.SearchService_SearchServer) error {
	searchLogger.Debug("Called search controller's Search method", "req", req)

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
		if err := srv.Send(&searchv1.SearchResponse{RecordCid: cid}); err != nil {
			return fmt.Errorf("failed to send record: %w", err)
		}
	}

	return nil
}
