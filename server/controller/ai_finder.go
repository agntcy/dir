// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/api/exportfmt"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	httpbodypb "google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var aiFinderLogger = logging.Logger("controller/ai-finder")

const (
	// wellKnownSpecVersion is the AI Catalog spec version embedded in
	// the served WellKnownCatalog payload.
	wellKnownSpecVersion = "1.0"

	// wellKnownHostDisplayName is the human-readable name for the
	// directory operator. Currently hardcoded; tracks catalogURNHost in
	// the data layer.
	//
	// TODO(ai-catalog): make this configurable once publisher data is
	// wired through the schema.
	wellKnownHostDisplayName = "AGNTCY Directory"
)

// aiFinderController adapts the AI Finder query language to the catalog query
// layer. GetWellKnownCatalog is served by the embedded Unimplemented server.
type aiFinderController struct {
	catalogv1.UnimplementedAIFinderServiceServer

	hostId string
	db     types.CatalogDatabaseAPI
	store  types.StoreAPI
	cfg    config.HTTPGatewayConfig
}

// NewAIFinderController returns an AIFinderServiceServer that serves the AI
// Catalog AI Finder surface. store may be nil — when omitted the ExportAgent
// RPC returns UNIMPLEMENTED (HTTP 501). All other RPCs remain functional.
func NewAIFinderController(hostId string, db types.CatalogDatabaseAPI, cfg config.HTTPGatewayConfig, store types.StoreAPI) catalogv1.AIFinderServiceServer {
	return &aiFinderController{
		db:     db,
		store:  store,
		cfg:    cfg,
		hostId: hostId,
	}
}

// ListAgents parses the filter, order, and paging arguments, queries the
// catalog, and returns one page of entries with a continuation token.
func (c *aiFinderController) ListAgents(ctx context.Context, req *catalogv1.ListAgentsRequest) (*catalogv1.ListAgentsResponse, error) {
	if req == nil {
		req = &catalogv1.ListAgentsRequest{}
	}

	aiFinderLogger.Debug("ListAgents called", "filter", req.GetFilter(), "order_by", req.GetOrderBy(), "page_size", req.GetPageSize())

	parsedFilter, err := parseAgentFilter(req.GetFilter())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
	}

	// publisherId is part of the grammar but not yet indexed.
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

	pageSize := int(clampPageSize(req.GetPageSize()))

	opts, ok := buildRecordFilterOptions(parsedFilter, order, pageSize, offset)
	if !ok {
		// type= matched no indexed module: zero rows, not an error.
		return &catalogv1.ListAgentsResponse{}, nil
	}

	entries, hasMore, err := c.db.GetCatalogEntries(opts...)
	if err != nil {
		aiFinderLogger.Error("failed to list catalog entries", "error", err)

		return nil, status.Error(codes.Internal, "failed to list catalog entries") //nolint:wrapcheck
	}

	if len(parsedFilter.Types) > 0 {
		entries = filterCatalogEntriesByMediaType(entries, parsedFilter.Types)
	}

	var nextPageToken string
	if hasMore {
		// Advance by the page size: the number of records consumed,
		// regardless of how many projected to an entry.
		nextPageToken = encodePageToken(offset + pageSize)
	}

	return &catalogv1.ListAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
	}, nil
}

// GetWellKnownCatalog returns a well-known catalog of agents. This is intended to be used
// for delegated discovery and returns catalog collections rather than catalog entries since
// there may be many entries.
func (c *aiFinderController) GetWellKnownCatalog(ctx context.Context, _ *catalogv1.GetWellKnownCatalogRequest) (*catalogv1.GetWellKnownCatalogResponse, error) {
	return &catalogv1.GetWellKnownCatalogResponse{
		Catalog: &catalogv1.WellKnownCatalog{
			SpecVersion: wellKnownSpecVersion,
			Host: &catalogv1.HostInfo{
				DisplayName: wellKnownHostDisplayName,
				Identifier:  new(c.cfg.PublicURL),
				TrustManifest: &catalogv1.TrustManifest{
					Identity:     catalogv1.GetCatalogUrnFor("host", c.hostId),
					IdentityType: new("did"),
				},
			},
			Collections: []*catalogv1.CatalogCollection{
				{
					DisplayName: "A2A Agents",
					Url:         c.collectionURL(catalogv1.ProtocolA2ACardJsonMediaType),
					Description: new("Agents that publish an A2A agent card."),
					MediaType:   new(catalogv1.ProtocolA2ACardJsonMediaType),
				},
				{
					DisplayName: "MCP Servers",
					Url:         c.collectionURL(catalogv1.ProtocolMCPCardJsonMediaType),
					Description: new("Agents that expose an MCP server connection."),
					MediaType:   new(catalogv1.ProtocolMCPCardJsonMediaType),
				},
				{
					DisplayName: "Agent Skills",
					Url:         c.collectionURL(catalogv1.ProtocolAgentSkillsMdMediaType),
					Description: new("Agents that publish a reusable Agent Skill definition as SKILL.md."),
					MediaType:   new(catalogv1.ProtocolAgentSkillsMdMediaType),
				},
				{
					DisplayName: "Agent Skill Bundles",
					Url:         c.collectionURL(catalogv1.ProtocolAgentSkillsBundleMediaType),
					Description: new("Agents that publish an Agent Skill directory bundled as .tar.gz."),
					MediaType:   new(catalogv1.ProtocolAgentSkillsBundleMediaType),
				},
			},
		},
	}, nil
}

// GetAgent returns a single CatalogEntry by CID.
func (c *aiFinderController) GetAgent(ctx context.Context, req *catalogv1.GetAgentRequest) (*catalogv1.GetAgentResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	cid := strings.TrimSpace(req.GetCid())
	if cid == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required") //nolint:wrapcheck
	}

	aiFinderLogger.Debug("GetAgent called", "cid", cid)

	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	entries, _, err := c.db.GetCatalogEntries(types.WithCIDs(cid), types.WithLimit(1))
	if err != nil {
		aiFinderLogger.Error("failed to load catalog entry", "cid", cid, "error", err)

		return nil, status.Error(codes.Internal, "failed to load catalog entry") //nolint:wrapcheck
	}

	if len(entries) == 0 {
		return nil, status.Errorf(codes.NotFound, "no catalog entry found for cid %q", cid)
	}

	return &catalogv1.GetAgentResponse{Entry: entries[0]}, nil
}

// ExportAgent pulls the full OASF record from the store and renders it in the
// requested format, returning raw bytes via google.api.HttpBody.
func (c *aiFinderController) ExportAgent(ctx context.Context, req *catalogv1.ExportAgentRequest) (*httpbodypb.HttpBody, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required") //nolint:wrapcheck
	}

	cid := strings.TrimSpace(req.GetCid())
	if cid == "" {
		return nil, status.Error(codes.InvalidArgument, "cid is required") //nolint:wrapcheck
	}

	formatName := strings.TrimSpace(req.GetFormat())
	if formatName == "" {
		formatName = exportfmt.FormatOASF
	}

	aiFinderLogger.Debug("ExportAgent called", "cid", cid, "format", formatName)

	if c.store == nil {
		return nil, status.Error(codes.Unimplemented, "agent export is not enabled on this registry") //nolint:wrapcheck
	}

	formatter, err := exportfmt.GetFormatter(formatName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported format %q; supported: %s",
			formatName, strings.Join(exportfmt.KnownFormats(), ", "))
	}

	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	record, err := c.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.Unknown {
			aiFinderLogger.Error("failed to pull record", "cid", cid, "error", err)

			return nil, status.Error(codes.Internal, "failed to retrieve agent") //nolint:wrapcheck
		}

		return nil, status.Error(st.Code(), st.Message()) //nolint:wrapcheck
	}

	if record.GetData() == nil {
		aiFinderLogger.Error("record has no data field", "cid", cid)

		return nil, status.Error(codes.Internal, "agent record is missing OASF data") //nolint:wrapcheck
	}

	data, err := formatter.Format(record)
	if err != nil {
		aiFinderLogger.Warn("failed to format record",
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

// collectionURL builds an absolute URL pointing at GET /v1/agents
// constrained to a single AI Catalog media type. The filter value is
// percent-encoded so the URL is safe to embed verbatim in JSON.
func (c *aiFinderController) collectionURL(mediaType string) string {
	base := strings.TrimRight(c.cfg.PublicURL, "/")

	v := url.Values{}
	v.Set("filter", "type="+mediaType)

	return fmt.Sprintf("%s/v1/agents?%s", base, v.Encode())
}
