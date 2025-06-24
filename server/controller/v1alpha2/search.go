// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1alpha2

import (
	"fmt"
	"strconv"
	"strings"

	searchtypes "github.com/agntcy/dir/api/search/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
)

var logger = logging.Logger("controller/search")

type searchCtlr struct {
	searchtypes.UnimplementedSearchServiceServer
	search types.SearchAPI
}

func NewSearchController(search types.SearchAPI) searchtypes.SearchServiceServer {
	return &searchCtlr{
		UnimplementedSearchServiceServer: searchtypes.UnimplementedSearchServiceServer{},
		search:                           search,
	}
}

func (c *searchCtlr) Search(req *searchtypes.SearchRequest, srv searchtypes.SearchService_SearchServer) error {
	logger.Debug("Called search controller's Search method", "req", req)

	filterOptions, err := queryToFilters(req)
	if err != nil {
		return fmt.Errorf("failed to create filter options: %w", err)
	}

	records, err := c.search.GetRecords(filterOptions...)
	if err != nil {
		return fmt.Errorf("failed to get records: %w", err)
	}

	for _, r := range records {
		if err := srv.Send(&searchtypes.SearchResponse{RecordCid: r.GetCID()}); err != nil {
			return fmt.Errorf("failed to send record: %w", err)
		}
	}

	return nil
}

func queryToFilters(req *searchtypes.SearchRequest) ([]types.FilterOption, error) { //nolint:gocognit,cyclop
	params := []types.FilterOption{
		types.WithLimit(int(req.GetLimit())),
		types.WithOffset(int(req.GetOffset())),
	}

	for _, query := range req.GetQueries() {
		switch query.GetType() {
		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
			logger.Warn("Unspecified query type, skipping", "query", query)

		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_NAME:
			params = append(params, types.WithName(query.GetValue()))

		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_VERSION:
			params = append(params, types.WithVersion(query.GetValue()))

		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_SKILL_ID:
			u64, err := strconv.ParseUint(query.GetValue(), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse skill ID %q: %w", query.GetValue(), err)
			}

			params = append(params, types.WithSkillIDs(u64))

		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME:
			params = append(params, types.WithSkillNames(query.GetValue()))

		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
			l := strings.SplitN(query.GetValue(), ":", 2) //nolint:mnd

			if len(l) == 1 && strings.TrimSpace(l[0]) != "" {
				params = append(params, types.WithLocatorTypes(l[0]))

				break
			}

			if len(l) == 2 { //nolint:mnd
				if strings.TrimSpace(l[0]) != "" {
					params = append(params, types.WithLocatorTypes(l[0]))
				}

				if strings.TrimSpace(l[1]) != "" {
					params = append(params, types.WithLocatorURLs(l[1]))
				}
			}

		case searchtypes.RecordQueryType_RECORD_QUERY_TYPE_EXTENSION:
			e := strings.SplitN(query.GetValue(), ":", 2) //nolint:mnd

			if len(e) == 1 && strings.TrimSpace(e[0]) != "" {
				params = append(params, types.WithExtensionNames(e[0]))

				break
			}

			if len(e) == 2 { //nolint:mnd
				if strings.TrimSpace(e[0]) != "" {
					params = append(params, types.WithExtensionNames(e[0]))
				}

				if strings.TrimSpace(e[1]) != "" {
					params = append(params, types.WithExtensionVersions(e[1]))
				}
			}

		default:
			logger.Warn("Unknown query type", "type", query.GetType())
		}
	}

	return params, nil
}
