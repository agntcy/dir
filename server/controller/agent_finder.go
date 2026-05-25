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

// AI Catalog Agent Finder API — see proto/agntcy/dir/catalog/v1/
// agent_finder_service.proto and §7 of the Agent Finder Specification.
//
// The controller is a thin adapter between the Agent Finder filter
// syntax (Appendix A) and the existing OASF record search. It:
//
//   1. Parses filter / order_by / page_size / page_token.
//   2. Translates the parsed filter into RecordFilters via FilterOption.
//   3. Hands the options to types.CatalogDatabaseAPI.GetCatalogEntries,
//      which queries records, loads signatures, and projects through
//      Record.ToCatalog.

var agentFinderLogger = logging.Logger("controller/agent-finder")

type agentFinderCtlr struct {
	catalogv1.UnimplementedAgentFinderServiceServer

	db types.CatalogDatabaseAPI
}

// NewAgentFinderController returns a catalogv1.AgentFinderServiceServer
// that serves the deterministic-browsing surface from the AI Catalog
// Agent Finder Specification (§7.2).
func NewAgentFinderController(db types.CatalogDatabaseAPI) catalogv1.AgentFinderServiceServer {
	return &agentFinderCtlr{db: db}
}

// ListAgents implements the GET /v1/agents endpoint.
//
// Maps directly to Agent Finder Specification §7.2 + Appendix A: parse
// the filter / order_by / page_size / page_token arguments, translate
// them into the existing record-search FilterOption surface, and let
// the data layer project records into CatalogEntries.
//
// All client-facing errors map to gRPC status codes per Appendix B;
// grpc-gateway translates them automatically (INVALID_ARGUMENT→400,
// UNIMPLEMENTED→501, INTERNAL→500).
func (c *agentFinderCtlr) ListAgents(ctx context.Context, req *catalogv1.ListAgentsRequest) (*catalogv1.ListAgentsResponse, error) {
	if req == nil {
		req = &catalogv1.ListAgentsRequest{}
	}

	agentFinderLogger.Debug("ListAgents called", "filter", req.GetFilter(), "order_by", req.GetOrderBy(), "page_size", req.GetPageSize())

	parsedFilter, err := parseAgentFilter(req.GetFilter())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
	}

	// publisherId is part of the spec but we don't yet store publisher
	// data. Returning empty results would silently misrepresent the
	// query; surface UNIMPLEMENTED (HTTP 501) so callers know to retry
	// without that clause until the schema lands.
	//
	// TODO(ai-catalog): wire publisher data through RecordFilters,
	// then drop this guard.
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

	pageSize := clampPageSize(req.GetPageSize())

	opts, ok := buildRecordFilterOptions(parsedFilter, order, int(pageSize), offset)
	if !ok {
		// type=… was set but every requested media type maps to an
		// OASF module this registry doesn't index. Per §3.3 the spec
		// is artifact-agnostic — unknown types are "zero rows", not
		// an error. Short-circuit without hitting the DB.
		return &catalogv1.ListAgentsResponse{}, nil
	}

	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	entries, hasMore, err := c.db.GetCatalogEntries(opts...)
	if err != nil {
		agentFinderLogger.Error("failed to list catalog entries", "error", err)

		return nil, status.Error(codes.Internal, "failed to list catalog entries") //nolint:wrapcheck
	}

	var nextPageToken string
	if hasMore {
		nextPageToken = encodePageToken(offset + len(entries))
	}

	return &catalogv1.ListAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
	}, nil
}

// buildRecordFilterOptions translates an Agent Finder filter +
// order + paging into the FilterOption surface that the existing OASF
// record search consumes.
func buildRecordFilterOptions(f agentFilter, order []orderByClause, pageSize int, offset int) ([]types.FilterOption, bool) {
	opts := []types.FilterOption{
		types.WithLimit(pageSize),
		types.WithOffset(offset),
	}

	// displayName → record name (case-insensitive substring).
	if f.DisplayName != "" {
		opts = append(opts, types.WithNames("*"+f.DisplayName+"*"))
	}

	// type=… → OASF module names.
	if len(f.Types) > 0 {
		var modules []string

		for _, mt := range f.Types {
			oasfName, ok := types.OASFModuleForMediaType(mt)
			if !ok {
				continue
			}

			modules = append(modules, oasfName)
		}

		if len(modules) == 0 {
			return nil, false
		}

		opts = append(opts, types.WithModuleNames(modules...))
	}

	// Both createdAfter and updatedAfter resolve to strict `>` comparisons
	// on records.oasf_created_at — the only OASF-supplied timestamp on a
	// record.
	if !f.CreatedAfter.IsZero() {
		opts = append(opts, types.WithCreatedAts(">"+f.CreatedAfter.UTC().Format(rfc3339UTC)))
	}

	if !f.UpdatedAfter.IsZero() {
		opts = append(opts, types.WithCreatedAts(">"+f.UpdatedAfter.UTC().Format(rfc3339UTC)))
	}

	// order_by → RecordOrderClause list.
	if len(order) > 0 {
		clauses := make([]types.RecordOrderClause, 0, len(order))
		for _, o := range order {
			clauses = append(clauses, types.RecordOrderClause{
				Column: o.Column,
				Desc:   o.Desc,
			})
		}

		opts = append(opts, types.WithOrderBy(clauses...))
	}

	return opts, true
}

// rfc3339UTC is the timestamp format used when emitting
// createdAfter / updatedAfter clauses to the data layer. Matches the
// spec's "ISO 8601 timestamp" wording (Appendix A).
const rfc3339UTC = "2006-01-02T15:04:05Z07:00"
