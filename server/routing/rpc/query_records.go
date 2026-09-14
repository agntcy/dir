// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/agntcy/dir/server/types"
	rpc "github.com/libp2p/go-libp2p-gorpc"
	"github.com/libp2p/go-libp2p/core/peer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DirServiceFuncQueryRecords = "QueryRecords"

	// MaxQueryRecords caps how many records one query call may return,
	// whatever limit the caller asks for.
	//
	// This is a unary call rather than a stream: gorpc's streaming path hands a
	// reflect.Value to the codec, and ugorji dropped its reflect.Value case in
	// v1.3.1, so streamed values serialise as empty structs.
	MaxQueryRecords = 1000
)

// RecordQuery is the wire form of a routing search query.
//
// routingv1.RecordQuery is a protobuf message and this transport is msgpack, so
// queries cross as plain structs rather than relying on reflection over
// generated protobuf internals.
type RecordQuery struct {
	// Type is the label namespace being queried, matching types.LabelType:
	// "skills", "domains", "modules" or "locators".
	Type string

	// Value is the label value without its namespace, e.g. "AI/ML".
	Value string
}

type QueryRecordsRequest struct {
	// Queries are OR'd: a record is returned if it matches any of them. The
	// caller scores and thresholds the results itself, which is why every
	// match ships its full label set.
	Queries []RecordQuery

	// Limit caps the returned records. Zero means MaxQueryRecords.
	Limit uint32
}

// RecordMatch is one record the peer holds that matched the query.
type RecordMatch struct {
	Cid string

	// Labels is the record's complete label set, namespaced, so the caller can
	// score it against the original queries without a second round trip.
	Labels []string
}

type QueryRecordsResponse struct {
	Records []RecordMatch

	// Truncated reports that the peer had more matches than the limit allowed.
	Truncated bool
}

// QueryRecords answers a peer's search over the records this node holds.
//
// It returns candidates, not decisions: matching is deliberately permissive
// (any query, prefix-inclusive) and the caller applies its own match score.
func (r *RPCAPI) QueryRecords(ctx context.Context, in *QueryRecordsRequest, out *QueryRecordsResponse) error {
	if in == nil || out == nil {
		return status.Error(codes.InvalidArgument, "invalid request: nil request/response") //nolint:wrapcheck
	}

	requestPeer, _ := rpc.GetRequestSender(ctx)

	logger.Debug("P2p RPC: Executing QueryRecords request on remote peer",
		"peer", r.service.localPeerID(),
		"request_peer", requestPeer,
		"queries", len(in.Queries),
		"limit", in.Limit,
	)

	if len(in.Queries) == 0 {
		return status.Error(codes.InvalidArgument, "at least one query is required") //nolint:wrapcheck
	}

	db, err := r.service.getDatabase()
	if err != nil {
		return err //nolint:wrapcheck
	}

	limit := int(min(in.Limit, MaxQueryRecords))
	if limit == 0 {
		limit = MaxQueryRecords
	}

	cids, truncated, err := matchingCIDs(db, in.Queries, limit)
	if err != nil {
		return err //nolint:wrapcheck
	}

	labels, err := db.GetRecordLabels(cids)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to load record labels: %v", err)
	}

	matches := make([]RecordMatch, 0, len(cids))

	for _, cid := range cids {
		recordLabels := labels[cid]
		if len(recordLabels) == 0 {
			// The record matched a label filter, so its labels vanished between
			// the two queries. Nothing useful to score against.
			continue
		}

		asStrings := make([]string, len(recordLabels))
		for i, label := range recordLabels {
			asStrings[i] = label.String()
		}

		matches = append(matches, RecordMatch{Cid: cid, Labels: asStrings})
	}

	logger.Debug("P2p RPC: QueryRecords served matches",
		"request_peer", requestPeer,
		"candidates", len(cids),
		"returned", len(matches),
		"truncated", truncated,
	)

	*out = QueryRecordsResponse{Records: matches, Truncated: truncated}

	return nil
}

// matchingCIDs runs each query separately and unions the results, reporting
// whether the limit cut the union short.
//
// One query per round trip because the filter API AND's different label kinds
// together, whereas routing search OR's its queries. Query counts are small.
func matchingCIDs(db types.DatabaseAPI, queries []RecordQuery, limit int) ([]string, bool, error) {
	seen := make(map[string]struct{}, limit)
	union := make([]string, 0, limit)

	for _, query := range queries {
		filters, err := queryFilters(query, limit)
		if err != nil {
			logger.Debug("Skipping unusable query", "type", query.Type, "value", query.Value, "error", err)

			continue
		}

		cids, err := db.GetRecordCIDs(filters...)
		if err != nil {
			return nil, false, status.Errorf(codes.Internal, "failed to query records: %v", err)
		}

		for _, cid := range cids {
			if _, ok := seen[cid]; ok {
				continue
			}

			seen[cid] = struct{}{}

			union = append(union, cid)

			if len(union) >= limit {
				return union, true, nil
			}
		}
	}

	return union, false, nil
}

// queryFilters translates one query into database filters.
//
// Hierarchical namespaces match the value itself or any descendant, so a query
// for "AI" finds "AI/ML" — the same prefix semantics the local matcher applies.
// Locators are flat and match exactly.
//
// Only published records are served. A peer arrives here through any single
// label this node advertised, so without the same filter that governs
// advertising, one shared label would expose every other record the node holds
// — including private pushes and ingested replicas.
func queryFilters(query RecordQuery, limit int) ([]types.FilterOption, error) {
	value := strings.TrimSpace(query.Value)
	if value == "" {
		return nil, fmt.Errorf("query value is empty")
	}

	descendants := value + "/*"
	base := []types.FilterOption{types.WithPublished(true), types.WithLimit(limit)}

	switch types.LabelType(query.Type) {
	case types.LabelTypeSkill:
		return append(base, types.WithSkillNames(value, descendants)), nil
	case types.LabelTypeDomain:
		return append(base, types.WithDomainNames(value, descendants)), nil
	case types.LabelTypeModule:
		return append(base, types.WithModuleNames(value, descendants)), nil
	case types.LabelTypeLocator:
		return append(base, types.WithLocatorTypes(value)), nil
	case types.LabelTypeUnknown:
		return nil, fmt.Errorf("unknown query type %q", query.Type)
	default:
		return nil, fmt.Errorf("unknown query type %q", query.Type)
	}
}

// QueryRecords asks a peer which of the records it holds match the queries.
func (s *Service) QueryRecords(ctx context.Context, peerID peer.ID, req *QueryRecordsRequest) ([]RecordMatch, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "query request is required") //nolint:wrapcheck
	}

	logger.Debug("P2p RPC: Executing QueryRecords request on remote peer", "peer", peerID, "queries", len(req.Queries))

	var resp QueryRecordsResponse

	err := s.rpcClient.CallContext(ctx, peerID, DirService, DirServiceFuncQueryRecords, req, &resp)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query records on peer %s: %v", peerID, err)
	}

	if resp.Truncated {
		logger.Warn("Peer truncated its query results", "peer", peerID, "returned", len(resp.Records))
	}

	return resp.Records, nil
}
