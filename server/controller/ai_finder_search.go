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

// SearchAgents answers a free-text query with relevance-ranked catalog entries.
//
// It is the API equivalent of `dirctl search "<query>"`: the query is decomposed
// into OASF skill, domain, and keyword signals, each signal is queried
// independently, and the union is ranked by how many signals matched each
// record. Both paths run the same nlsearch fan-out over the same signals, so the
// same query returns the same records in the same order — the CLI reaches the
// search layer over gRPC, this handler queries the database in-process.
//
// Ranking happens over the whole candidate set before paging, so a page is a
// slice of an already-ordered list.
func (c *aiFinderController) SearchAgents(ctx context.Context, req *catalogv1.SearchAgentsRequest) (*catalogv1.SearchAgentsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	query := strings.TrimSpace(req.GetQuery())
	if query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required") //nolint:wrapcheck
	}

	// Log the length rather than the text: the query is user-supplied free form
	// and may contain sensitive terms.
	aiFinderLogger.Debug("SearchAgents called", "query_len", len(query), "page_size", req.GetPageSize())

	offset, err := decodePageToken(req.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
	}

	pageSize := int(clampPageSize(req.GetPageSize()))

	if c.ext == nil {
		return nil, status.Error(codes.Unavailable, //nolint:wrapcheck
			"natural-language search is unavailable: no OASF extractor is configured on this gateway")
	}

	signals, err := nlsearch.Decompose(ctx, query, c.ext, extractor.ExtractOptions{})
	if err != nil {
		aiFinderLogger.Error("failed to extract search signals", "query_len", len(query), "error", err)

		return nil, status.Error(codes.Unavailable, "natural-language search is temporarily unavailable") //nolint:wrapcheck
	}

	if len(signals) == 0 {
		// Nothing to search on. An empty page is the honest answer: unlike the
		// CLI, which can tell the user to rephrase, this returns a normal result.
		return &catalogv1.SearchAgentsResponse{}, nil
	}

	ranked := nlsearch.FanOutAndScore(ctx, signals, recordSearcher{db: c.db}, nlsearch.FanOutOptions{})

	// A failed signal degrades recall rather than the request; surface it in the
	// logs so a partly-empty ranking is explainable.
	for _, hit := range ranked.PerSignal {
		if hit.Err != nil {
			aiFinderLogger.Warn("search signal failed", "signal_type", hit.Signal.Type.String(), "error", hit.Err)
		}
	}

	page := pageCIDs(ranked.CIDs, pageSize, offset)

	entries, err := c.hydrateInRankOrder(page)
	if err != nil {
		return nil, err
	}

	var nextPageToken string
	if offset+pageSize < len(ranked.CIDs) {
		nextPageToken = encodePageToken(offset + pageSize)
	}

	return &catalogv1.SearchAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
		TotalCount:    uint32(len(ranked.CIDs)), //nolint:gosec
	}, nil
}

// pageCIDs slices one page out of the ranked CIDs, tolerating an offset past the
// end (which yields an empty page rather than an error).
func pageCIDs(cids []string, pageSize, offset int) []string {
	if offset >= len(cids) {
		return nil
	}

	end := min(offset+pageSize, len(cids))

	return cids[offset:end]
}

// recordSearcher adapts the server database to nlsearch.Searcher, so the gateway
// runs the same fan-out as the CLI without a network hop.
type recordSearcher struct {
	db aiFinderDatabaseAPI
}

func (s recordSearcher) SearchCIDs(_ context.Context, query *searchv1.RecordQuery, limit uint32) ([]string, error) {
	filterOpts, err := databaseutils.QueryToFilters([]*searchv1.RecordQuery{query})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to build search filters") //nolint:wrapcheck
	}

	// Match the CLI's per-signal query: recency-ordered, capped. Relevance is
	// decided by the scorer across signals, not by any single query's order.
	filterOpts = append(filterOpts,
		types.WithLimit(int(limit)), //nolint:gosec
		sortModeToOrderBy(searchv1.SortMode_SORT_MODE_RECENCY),
	)

	cids, err := s.db.GetRecordCIDs(filterOpts...)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to search record CIDs") //nolint:wrapcheck
	}

	return cids, nil
}

// hydrateInRankOrder projects ranked CIDs into catalog entries, preserving the
// rank order. Records that do not project to an entry are absent, so a page may
// hold fewer entries than CIDs.
func (c *aiFinderController) hydrateInRankOrder(cids []string) ([]*catalogv1.CatalogEntry, error) {
	if len(cids) == 0 {
		return nil, nil
	}

	entries, _, err := c.db.GetCatalogEntries(
		types.WithCIDs(cids...),
		types.WithLimit(len(cids)),
	)
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

// catalogEntryCID recovers the record CID from a catalog entry's identifier URN
// ("urn:ai:<host>:cid:<cid>"). Identifiers that are not URNs (e.g. bare CIDs in
// tests) are returned unchanged.
func catalogEntryCID(entry *catalogv1.CatalogEntry) string {
	id := entry.GetIdentifier()
	if _, after, ok := strings.Cut(id, ":cid:"); ok {
		return after
	}

	return id
}
