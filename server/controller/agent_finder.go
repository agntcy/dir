// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	httpbodypb "google.golang.org/genproto/googleapis/api/httpbody"
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

	// store is the content-addressable store used by GetAgent to pull
	// full OASF record documents. Nil-safe: when nil, GetAgent returns
	// UNIMPLEMENTED so deployments that wire only the listing surface
	// degrade cleanly rather than panicking on a request.
	store types.StoreAPI

	// publicBaseURL is the absolute scheme+authority URL clients can
	// reach this directory at, e.g. "http://localhost:8889".
	publicBaseURL string
}

// NewAgentFinderController returns a catalogv1.AgentFinderServiceServer
// that serves the AI Catalog Agent Finder surface (§7.2 list + RFC 8615
// well-known catalog + per-record retrieval by CID).
//
// store may be nil — when omitted the GetAgent RPC returns UNIMPLEMENTED
// (HTTP 501). All other RPCs remain functional.
func NewAgentFinderController(db types.CatalogDatabaseAPI, store types.StoreAPI, publicBaseURL string) catalogv1.AgentFinderServiceServer {
	return &agentFinderCtlr{
		db:            db,
		store:         store,
		publicBaseURL: publicBaseURL,
	}
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

// GetAgent implements the GET /v1/agents/{cid} endpoint.
//
// Returns the single CatalogEntry for the given CID — the same shape
// a client would have seen for this record inside
// ListAgentsResponse.results. That symmetry is the point: a UI that
// lists, picks a row, and drills in by CID gets the exact object it
// already had, just on its own URL (and own cache key).
//
// The full OASF record (or any other dirctl export format) lives on
// the sub-resource handled by ExportAgent — different vocabulary,
// different caching semantics, different endpoint.
//
// Error mapping (Appendix B):
//   - InvalidArgument (400): empty / whitespace-only CID. Length is
//     already capped by buf-validate before we get here.
//   - NotFound       (404): no catalog entry exists for this CID,
//     either because the record was never indexed or because it has
//     no AI Catalog projection (e.g. zero known modules).
//   - Internal       (500): backing database failure.
func (c *agentFinderCtlr) GetAgent(ctx context.Context, req *catalogv1.GetAgentRequest) (*catalogv1.GetAgentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	cid := strings.TrimSpace(req.GetCid())
	if cid == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required") //nolint:wrapcheck
	}

	agentFinderLogger.Debug("GetAgent called", "cid", cid)

	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	// One row, by primary key. The catalog projection path
	// (GetCatalogEntries → Record.ToCatalog) is the same one that
	// ListAgents uses, so consumers see consistent fields across the
	// list and detail endpoints.
	entries, _, err := c.db.GetCatalogEntries(types.WithCIDs(cid), types.WithLimit(1))
	if err != nil {
		agentFinderLogger.Error("failed to load catalog entry", "cid", cid, "error", err)

		return nil, status.Error(codes.Internal, "failed to load catalog entry") //nolint:wrapcheck
	}

	if len(entries) == 0 {
		return nil, status.Errorf(codes.NotFound, "no catalog entry found for cid %q", cid)
	}

	return &catalogv1.GetAgentResponse{Entry: entries[0]}, nil
}

// ExportAgent implements the GET /v1/agents/{cid}/export endpoint.
//
// Pulls the full record from the content-addressable store and runs
// it through the requested formatter in api/exportfmt, returning the
// raw bytes in a google.api.HttpBody so the HTTP gateway can write
// them to the response body with the right Content-Type — byte-
// identical to what `dirctl export --format=<X>` would write.
//
// Format defaults to "oasf" when the query parameter is omitted, so
// `curl /v1/agents/{cid}/export` does the obvious thing.
//
// Error mapping:
//   - InvalidArgument    (400): empty / whitespace-only CID, or a
//     format that isn't registered in api/exportfmt.
//   - FailedPrecondition (400): the record exists but cannot be
//     projected to the requested format (e.g. asking for "agent-skill"
//     on a record without core/language_model/agentskills, "a2a" on a
//     record without integration/a2a, etc.). The request was valid;
//     the data simply doesn't carry what the format reads.
//   - NotFound           (404): the store has no record with this CID.
//   - Internal           (500): store error, or genuinely unexpected
//     formatter failure (e.g. JSON marshal). NOT format/data
//     mismatches — those are FailedPrecondition above.
//   - Unimplemented      (501): no StoreAPI was wired into the controller.
func (c *agentFinderCtlr) ExportAgent(ctx context.Context, req *catalogv1.ExportAgentRequest) (*httpbodypb.HttpBody, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	cid := strings.TrimSpace(req.GetCid())
	if cid == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required") //nolint:wrapcheck
	}

	// Default empty → oasf so the URL stays curl-friendly. The proto
	// already allow-lists the legal values; we only have to handle
	// the "" → "oasf" expansion here.
	formatName := strings.TrimSpace(req.GetFormat())
	if formatName == "" {
		formatName = exportfmt.FormatOASF
	}

	agentFinderLogger.Debug("ExportAgent called", "cid", cid, "format", formatName)

	if c.store == nil {
		// Defensive: a misconfigured deployment that wires the
		// AgentFinder controller without a backing store should not
		// silently 404 the export sub-resource. Surface the
		// misconfiguration explicitly so operators see it in client
		// logs and traces.
		return nil, status.Error(codes.Unimplemented, "agent export is not enabled on this registry") //nolint:wrapcheck
	}

	formatter, err := exportfmt.GetFormatter(formatName)
	if err != nil {
		// Mirrors `dirctl export --format=...` behaviour. The proto
		// allow-list catches typos at the validation layer; this
		// guard covers the case where a format is listed in the
		// proto enum but not yet registered (defence in depth).
		return nil, status.Errorf(codes.InvalidArgument, "unsupported format %q; supported: %s",
			formatName, strings.Join(exportfmt.KnownFormats(), ", "))
	}

	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	record, err := c.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		// Preserve the gRPC code from the store layer (OCI returns
		// NotFound for unknown CIDs; other failures bubble up as
		// Internal). Mirrors StoreService.Pull behaviour for
		// consistent codes across retrieval paths.
		st := status.Convert(err)
		if st.Code() == codes.Unknown {
			agentFinderLogger.Error("failed to pull record", "cid", cid, "error", err)

			return nil, status.Error(codes.Internal, "failed to retrieve agent") //nolint:wrapcheck
		}

		return nil, status.Error(st.Code(), st.Message()) //nolint:wrapcheck
	}

	// Pre-check: every formatter rejects nil data with a generic
	// "record contains no data" error. Catching it here gives the
	// HTTP client a clearer message (the issue is server-side data
	// integrity, not the choice of format).
	if record.GetData() == nil {
		agentFinderLogger.Error("record has no data field", "cid", cid)

		return nil, status.Error(codes.Internal, "agent record is missing OASF data") //nolint:wrapcheck
	}

	data, err := formatter.Format(record)
	if err != nil {
		// Split format failures into two buckets:
		//
		//  - ErrUnsupportedRecord  → FailedPrecondition (HTTP 400).
		//    The request was well-formed and the format is registered,
		//    but this particular record lacks the OASF module the
		//    format projects from (e.g. asking for "agent-skill" on a
		//    record without core/language_model/agentskills). That's a
		//    client/data mismatch, not a server fault — 5xx would
		//    falsely page operators on a routine client error.
		//
		//  - anything else         → Internal (HTTP 500).
		//    JSON marshaling failures and other unexpected errors stay
		//    Internal because they indicate something is wrong on the
		//    server side, not with the request shape or the data.
		//
		// We log the underlying error in both branches so operators
		// can see the translator's detail; only Internal escalates
		// to a 5xx on the wire.
		agentFinderLogger.Warn("failed to format record",
			"cid", cid, "format", formatName, "error", err)

		if errors.Is(err, exportfmt.ErrUnsupportedRecord) {
			return nil, status.Errorf(codes.FailedPrecondition,
				"record %q cannot be exported in %q format: %s",
				cid, formatName, err.Error())
		}

		return nil, status.Errorf(codes.Internal, "failed to render agent in %s format", formatName) //nolint:wrapcheck
	}

	return &httpbodypb.HttpBody{
		ContentType: exportfmt.ContentTypeForExtension(formatter.FileExtension()),
		Data:        data,
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
