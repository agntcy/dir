// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/org"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var searchLogger = logging.Logger("controller/search")

type searchCtlr struct {
	searchv1.UnimplementedSearchServiceServer
	db          types.DatabaseAPI
	store       types.StoreAPI
	orgResolver org.Resolver // optional; nil when not configured
}

// SearchControllerOption configures the search controller.
type SearchControllerOption func(*searchCtlr)

// WithOrgResolver attaches an org resolver to the search controller,
// enabling --manager queries.
func WithOrgResolver(r org.Resolver) SearchControllerOption {
	return func(c *searchCtlr) {
		c.orgResolver = r
	}
}

func NewSearchController(db types.DatabaseAPI, store types.StoreAPI, opts ...SearchControllerOption) searchv1.SearchServiceServer {
	c := &searchCtlr{
		UnimplementedSearchServiceServer: searchv1.UnimplementedSearchServiceServer{},
		db:                               db,
		store:                            store,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// expandManagerQueries separates RECORD_QUERY_TYPE_MANAGER queries from the rest,
// expands each manager alias via the org resolver, and returns the remaining queries
// plus an OwnerAliases filter option. Returns an error if a manager query is present
// but no resolver is configured.
func (c *searchCtlr) expandManagerQueries(queries []*searchv1.RecordQuery) ([]*searchv1.RecordQuery, []types.FilterOption, error) {
	var remaining []*searchv1.RecordQuery

	var ownerAliases []string

	for _, q := range queries {
		if q.GetType() != searchv1.RecordQueryType_RECORD_QUERY_TYPE_MANAGER {
			remaining = append(remaining, q)

			continue
		}

		if c.orgResolver == nil {
			return nil, nil, fmt.Errorf("--manager query requires an org resolver to be configured on the server")
		}

		aliases, err := c.orgResolver.Expand(nil, q.GetValue()) //nolint:staticcheck
		if err != nil {
			return nil, nil, fmt.Errorf("expanding manager %q: %w", q.GetValue(), err)
		}

		ownerAliases = append(ownerAliases, aliases...)
	}

	var extra []types.FilterOption
	if len(ownerAliases) > 0 {
		extra = append(extra, types.WithOwnerAliases(ownerAliases...))
	}

	return remaining, extra, nil
}

func (c *searchCtlr) SearchCIDs(req *searchv1.SearchCIDsRequest, srv searchv1.SearchService_SearchCIDsServer) error {
	searchLogger.Debug("Called search controller's SearchCIDs method", "req", req)

	remaining, extra, err := c.expandManagerQueries(req.GetQueries())
	if err != nil {
		return err
	}

	filterOptions, err := databaseutils.QueryToFilters(remaining)
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	filterOptions = append(filterOptions, extra...)
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

func (c *searchCtlr) SearchRecords(req *searchv1.SearchRecordsRequest, srv searchv1.SearchService_SearchRecordsServer) error {
	searchLogger.Debug("Called search controller's SearchRecords method", "req", req)

	remaining, extra, err := c.expandManagerQueries(req.GetQueries())
	if err != nil {
		return err
	}

	filterOptions, err := databaseutils.QueryToFilters(remaining)
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	filterOptions = append(filterOptions, extra...)
	filterOptions = append(filterOptions,
		types.WithLimit(int(req.GetLimit())),
		types.WithOffset(int(req.GetOffset())),
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
