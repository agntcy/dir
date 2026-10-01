// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var localLogger = logging.Logger("routing/local")

// errNoDatabase reports that routing was constructed without the SQL index it
// needs to answer what this node holds.
var errNoDatabase = errors.New("routing has no database")

// routeLocal answers questions about the records this node holds.
//
// Held records and their labels come from the SQL index, which the ingest path
// maintains for content arriving from any source and which SearchService
// queries too.
type routeLocal struct {
	db types.DatabaseAPI
}

func newLocal(db types.DatabaseAPI) *routeLocal {
	return &routeLocal{db: db}
}

// List returns the records this node has published, filtered by the request's
// queries. Records that are merely held are absent: nothing announces them.
//
// Queries AND together: a record is returned only if it satisfies every one.
func (r *routeLocal) List(ctx context.Context, req *routingv1.ListRequest) (<-chan *routingv1.ListResponse, error) {
	localLogger.Debug("Called local routing's List method", "req", req)

	if r.db == nil {
		return nil, status.Error(codes.Unavailable, "local routing has no database to list from") //nolint:wrapcheck
	}

	// Duplicate queries would otherwise cost a redundant round trip each.
	originalQueries := req.GetQueries()
	queries := deduplicateQueries(originalQueries)

	if len(originalQueries) != len(queries) {
		localLogger.Info("Deduplicated list queries for consistent filtering",
			"originalCount", len(originalQueries), "deduplicatedCount", len(queries))
	}

	outCh := make(chan *routingv1.ListResponse)

	go func() {
		defer close(outCh)

		r.listLocalRecords(ctx, queries, req.GetLimit(), outCh)
	}()

	return outCh, nil
}

// listLocalRecords resolves the query to a CID set, loads each record's labels
// and streams the results.
func (r *routeLocal) listLocalRecords(ctx context.Context, queries []*routingv1.RecordQuery, limit uint32, outCh chan<- *routingv1.ListResponse) {
	cids, err := r.matchingCIDs(queries, int(limit))
	if err != nil {
		localLogger.Error("Failed to list local records", "error", err)

		return
	}

	if len(cids) == 0 {
		localLogger.Debug("Completed List operation", "processed", 0, "queries", len(queries))

		return
	}

	labels, err := r.db.GetRecordLabels(cids)
	if err != nil {
		localLogger.Error("Failed to load labels for local records", "error", err)

		return
	}

	sent := 0

	for _, cid := range cids {
		recordLabels := labels[cid]

		asStrings := make([]string, len(recordLabels))
		for i, label := range recordLabels {
			asStrings[i] = label.String()
		}

		select {
		case outCh <- &routingv1.ListResponse{
			RecordRef: &corev1.RecordRef{Cid: cid},
			Labels:    asStrings,
		}:
			sent++
		case <-ctx.Done():
			localLogger.Debug("List cancelled", "sent", sent)

			return
		}
	}

	localLogger.Debug("Completed List operation", "processed", sent, "queries", len(queries))
}

// matchingCIDs returns the CIDs of held records satisfying every query.
//
// Each query runs on its own and the results are intersected, because the
// filter API can only AND across label kinds: two skill queries in one filter
// set would OR together and wrongly widen the result.
func (r *routeLocal) matchingCIDs(queries []*routingv1.RecordQuery, limit int) ([]string, error) {
	if len(queries) <= 1 {
		filters, err := listFilters(queries, limit)
		if err != nil {
			return nil, err
		}

		cids, err := r.db.GetRecordCIDs(filters...)
		if err != nil {
			return nil, fmt.Errorf("failed to query records: %w", err)
		}

		return cids, nil
	}

	var matched []string

	for i, query := range queries {
		// The limit cannot be pushed down here: a record ranked past it in one
		// query could still belong in the intersection.
		filters, err := listFilters([]*routingv1.RecordQuery{query}, maxListCandidates)
		if err != nil {
			return nil, err
		}

		cids, err := r.db.GetRecordCIDs(filters...)
		if err != nil {
			return nil, fmt.Errorf("failed to query records: %w", err)
		}

		if i == 0 {
			matched = cids

			continue
		}

		matched = intersect(matched, cids)
		if len(matched) == 0 {
			return nil, nil
		}
	}

	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}

	return matched, nil
}

// baseListFilters apply to every List, whatever the request asks for.
// Unpublished records are excluded because List reports what this node is
// providing, not everything it holds.
var baseListFilters = []types.FilterOption{types.WithPublished(true)}

// listFilters translates the queries into database filters. An empty query set
// selects every published record, which is what an unfiltered List asks for.
func listFilters(queries []*routingv1.RecordQuery, limit int) ([]types.FilterOption, error) {
	filters := make([]types.FilterOption, 0, len(queries)+len(baseListFilters)+1)
	filters = append(filters, baseListFilters...)

	for _, query := range queries {
		filter, err := listFilter(query)
		if err != nil {
			return nil, err
		}

		filters = append(filters, filter)
	}

	if limit > 0 {
		filters = append(filters, types.WithLimit(limit))
	}

	return filters, nil
}

// listFilter translates one query into a database filter.
//
// Hierarchical namespaces match the value or any descendant, so a query for
// "AI" finds "AI/ML". Locators are flat and match exactly.
func listFilter(query *routingv1.RecordQuery) (types.FilterOption, error) {
	value := query.GetValue()
	if value == "" {
		return nil, fmt.Errorf("query of type %s has no value", query.GetType())
	}

	descendants := value + "/*"

	switch query.GetType() {
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL:
		return types.WithSkillNames(value, descendants), nil
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN:
		return types.WithDomainNames(value, descendants), nil
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE:
		return types.WithModuleNames(value, descendants), nil
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
		return types.WithLocatorTypes(value), nil
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
		return nil, fmt.Errorf("query type is unspecified")
	default:
		return nil, fmt.Errorf("unknown query type %s", query.GetType())
	}
}

// intersect returns the members of left that also appear in right, preserving
// left's ordering.
func intersect(left, right []string) []string {
	set := make(map[string]struct{}, len(right))
	for _, cid := range right {
		set[cid] = struct{}{}
	}

	kept := left[:0]

	for _, cid := range left {
		if _, ok := set[cid]; ok {
			kept = append(kept, cid)
		}
	}

	return kept
}
