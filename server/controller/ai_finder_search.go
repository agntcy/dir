// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	databaseutils "github.com/agntcy/dir/server/database/utils"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/extractor"
	"github.com/agntcy/dir/utils/nlsearch"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// searchExtractorTiers is how many score tiers of skills/domains the gateway
// considers per query. 2 (the two closest groups) widens recall over the
// extractor's default of 1, at the cost of some precision.
const searchExtractorTiers = 2

// SearchAgents answers a free-text natural-language query with relevance-ranked
// catalog entries (extract-then-filter, v1):
//
//  1. extract top-tier OASF skills and domains from the query;
//  2. content-search on them (SearchService's GetRecordCIDs, in-process);
//  3. hydrate the CIDs into CatalogEntry rows, applying the optional facet
//     filter, preserving the search order.
//
// When extraction yields no skill/domain signal it falls back to a displayName
// substring match so the user still gets results. Facets (type, verified,
// trusted, safe, tags) reuse the ListAgents filter grammar and compose with the
// query.
func (c *aiFinderController) SearchAgents(ctx context.Context, req *catalogv1.SearchAgentsRequest) (*catalogv1.SearchAgentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required") //nolint:wrapcheck
	}

	aiFinderLogger.Debug("SearchAgents called", "query", query, "filter", req.GetFilter(), "page_size", req.GetPageSize())

	parsedFilter, err := parseAgentFilter(req.GetFilter())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
	}

	// publisherId is part of the grammar but not yet indexed.
	if len(parsedFilter.PublisherIDs) > 0 {
		return nil, status.Error(codes.Unimplemented, "publisherId filter is not yet supported") //nolint:wrapcheck
	}

	facetOpts, ok := buildCatalogFacetOptions(parsedFilter)
	if !ok {
		// type= matched no indexed module: zero rows, not an error.
		return &catalogv1.SearchAgentsResponse{}, nil
	}

	offset, err := decodePageToken(req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
	}

	pageSize := int(clampPageSize(req.GetPageSize()))

	if c.ext == nil {
		return nil, status.Error(codes.Unavailable, //nolint:wrapcheck
			"natural-language search is unavailable: no OASF extractor is configured on this gateway")
	}

	signals, err := nlsearch.Decompose(ctx, query, c.ext, extractor.ExtractOptions{Tiers: searchExtractorTiers})
	if err != nil {
		aiFinderLogger.Error("failed to extract search signals", "query", query, "error", err)

		return nil, status.Error(codes.Unavailable, "natural-language search is temporarily unavailable") //nolint:wrapcheck
	}

	queries := recordQueriesFromSignals(signals)
	if len(queries) == 0 {
		// Empty extraction: degrade to a displayName substring match.
		return c.searchByName(query, facetOpts, pageSize, offset)
	}

	return c.searchByQueries(queries, facetOpts, pageSize, offset)
}

// recordQueriesFromSignals keeps the taxonomy signals (skills, domains) and maps
// them to RecordQuery values. Keyword signals are intentionally dropped: the v1
// gateway path is extract-then-filter, not the CLI's keyword fan-out.
func recordQueriesFromSignals(signals []nlsearch.Signal) []*searchv1.RecordQuery {
	var queries []*searchv1.RecordQuery

	for _, s := range signals {
		if s.Type == nlsearch.SignalTypeSkillName || s.Type == nlsearch.SignalTypeDomainName {
			queries = append(queries, &searchv1.RecordQuery{
				Type:  s.QueryType(),
				Value: s.Value,
			})
		}
	}

	return queries
}

// searchByQueries runs a single relevance-sorted content search over the
// extracted skill/domain queries, then hydrates the resulting CIDs into catalog
// entries with the facet filter applied, preserving the search order.
func (c *aiFinderController) searchByQueries(queries []*searchv1.RecordQuery, facetOpts []types.CatalogQueryOption, pageSize, offset int) (*catalogv1.SearchAgentsResponse, error) {
	filterOpts, err := databaseutils.QueryToFilters(queries)
	if err != nil {
		aiFinderLogger.Error("failed to build search filters", "error", err)

		return nil, status.Error(codes.Internal, "failed to search catalog") //nolint:wrapcheck
	}

	// Peek one past the page to learn whether a next page exists.
	filterOpts = append(filterOpts,
		types.WithLimit(pageSize+1),
		types.WithOffset(offset),
		sortModeToOrderBy(searchv1.SortMode_SORT_MODE_RELEVANCE),
	)

	cids, err := c.db.GetRecordCIDs(filterOpts...)
	if err != nil {
		aiFinderLogger.Error("failed to search record CIDs", "error", err)

		return nil, status.Error(codes.Internal, "failed to search catalog") //nolint:wrapcheck
	}

	hasMore := len(cids) > pageSize
	if hasMore {
		cids = cids[:pageSize]
	}

	entries, err := c.hydrateInCIDOrder(cids, facetOpts)
	if err != nil {
		return nil, err
	}

	var nextPageToken string
	if hasMore {
		nextPageToken = encodePageToken(offset + pageSize)
	}

	return &catalogv1.SearchAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
	}, nil
}

// searchByName is the empty-extraction fallback: a displayName substring match
// over the catalog, composing with the facet filter.
func (c *aiFinderController) searchByName(query string, facetOpts []types.CatalogQueryOption, pageSize, offset int) (*catalogv1.SearchAgentsResponse, error) {
	opts := append([]types.CatalogQueryOption{
		types.WithNames("*" + query + "*"),
		types.WithLimit(pageSize),
		types.WithOffset(offset),
	}, facetOpts...)

	entries, hasMore, err := c.db.GetCatalogEntries(opts...)
	if err != nil {
		aiFinderLogger.Error("failed to search catalog by name", "error", err)

		return nil, status.Error(codes.Internal, "failed to search catalog") //nolint:wrapcheck
	}

	var nextPageToken string
	if hasMore {
		nextPageToken = encodePageToken(offset + pageSize)
	}

	return &catalogv1.SearchAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
	}, nil
}

// hydrateInCIDOrder projects the given CIDs into catalog entries with the facet
// filter applied, returned in the same order as cids. Entries dropped by a facet
// (or unprojectable records) are simply absent, so a facet-narrowed page may
// hold fewer than len(cids) entries — an accepted v1 trade-off.
func (c *aiFinderController) hydrateInCIDOrder(cids []string, facetOpts []types.CatalogQueryOption) ([]*catalogv1.CatalogEntry, error) {
	if len(cids) == 0 {
		return nil, nil
	}

	opts := append([]types.CatalogQueryOption{
		types.WithCIDs(cids...),
		types.WithLimit(len(cids)),
	}, facetOpts...)

	entries, _, err := c.db.GetCatalogEntries(opts...)
	if err != nil {
		aiFinderLogger.Error("failed to hydrate search results", "error", err)

		return nil, status.Error(codes.Internal, "failed to hydrate search results") //nolint:wrapcheck
	}

	byCID := make(map[string]*catalogv1.CatalogEntry, len(entries))
	for _, e := range entries {
		byCID[catalogEntryCID(e)] = e
	}

	ordered := make([]*catalogv1.CatalogEntry, 0, len(cids))

	for _, cid := range cids {
		if e, ok := byCID[cid]; ok {
			ordered = append(ordered, e)
		}
	}

	return ordered, nil
}

// catalogEntryCID recovers the record CID from a catalog entry's identifier
// URN ("urn:ai:<host>:cid:<cid>"). Identifiers that are not URNs (e.g. bare CIDs
// in tests) are returned unchanged.
func catalogEntryCID(entry *catalogv1.CatalogEntry) string {
	id := entry.GetIdentifier()
	if _, after, ok := strings.Cut(id, ":cid:"); ok {
		cid, _, _ := strings.Cut(after, ":")

		return cid
	}

	return id
}
