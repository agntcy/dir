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
	"github.com/agntcy/dir/utils/extractor"
	"github.com/agntcy/dir/utils/logging"
	httpbodypb "google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var aiFinderLogger = logging.Logger("controller/ai-finder")

const (
	// wellKnownSpecVersion is the AI Catalog spec version embedded in
	// the served well-known AICatalog payload.
	wellKnownSpecVersion = "1.0"

	// wellKnownHostDisplayName is the human-readable name for the
	// directory operator. Currently hardcoded; tracks catalogURNHost in
	// the data layer.
	//
	// TODO(ai-catalog): make this configurable once publisher data is
	// wired through the schema.
	wellKnownHostDisplayName = "AGNTCY Directory"
)

// aiFinderDatabaseAPI is the database surface the AI Finder controller needs:
// the catalog query layer (browsing, hydration) plus the record-CID content
// search that SearchAgents fans out over. The concrete server database
// satisfies it; the non-search RPCs only exercise the catalog half.
type aiFinderDatabaseAPI interface {
	types.CatalogDatabaseAPI
	GetRecordCIDs(opts ...types.FilterOption) ([]string, error)
}

// aiFinderController adapts the AI Finder query language to the catalog query
// layer.
type aiFinderController struct {
	catalogv1.UnimplementedAIFinderServiceServer

	hostId string
	db     aiFinderDatabaseAPI
	gate   types.RecordGate
	store  types.StoreAPI
	cfg    config.HTTPGatewayConfig
	ext    extractor.Extractor
}

// AIFinderOption configures optional dependencies of the AI Finder controller.
type AIFinderOption func(*aiFinderController)

// WithExtractor wires an OASF extractor resolved by the server, enabling the
// extractor-backed RPCs. Without it ExtractTaxonomy and SearchAgents return
// UNAVAILABLE (HTTP 503); all other RPCs remain functional.
func WithExtractor(ext extractor.Extractor) AIFinderOption {
	return func(c *aiFinderController) {
		c.ext = ext
	}
}

// NewAIFinderController returns an AIFinderServiceServer that serves the AI
// Catalog AI Finder surface. store may be nil — when omitted the ExportAgent
// RPC returns UNIMPLEMENTED (HTTP 501). All other RPCs remain functional.
func NewAIFinderController(hostId string, db aiFinderDatabaseAPI, gate types.RecordGate, cfg config.HTTPGatewayConfig, store types.StoreAPI, opts ...AIFinderOption) catalogv1.AIFinderServiceServer {
	c := &aiFinderController{
		db:     db,
		gate:   gate,
		store:  store,
		cfg:    cfg,
		hostId: hostId,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ListAgents parses the filter, order, and paging arguments, queries the
// catalog, and returns one page of entries with a continuation token.
func (c *aiFinderController) ListAgents(ctx context.Context, req *catalogv1.ListAgentsRequest) (*catalogv1.ListAgentsResponse, error) {
	if req == nil {
		req = &catalogv1.ListAgentsRequest{}
	}

	aiFinderLogger.Debug("ListAgents called", "filter", req.GetFilter(), "order_by", req.GetOrderBy(), "page_size", req.GetPageSize())

	entries, nextPageToken, totalCount, err := c.listCatalogPage(ctx, req.GetFilter(), req.GetOrderBy(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}

	return &catalogv1.ListAgentsResponse{
		Results:       entries,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// ListTags returns distinct catalog tags derived from OASF skills, domains,
// and record annotations.
func (c *aiFinderController) ListTags(ctx context.Context, _ *catalogv1.ListTagsRequest) (*catalogv1.ListTagsResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	aiFinderLogger.Debug("ListTags called")

	tags, err := c.db.ListCatalogTags()
	if err != nil {
		aiFinderLogger.Error("failed to list catalog tags", "error", err)

		return nil, status.Error(codes.Internal, "failed to list catalog tags") //nolint:wrapcheck
	}

	return &catalogv1.ListTagsResponse{Tags: tags}, nil
}

// GetWellKnownCatalog returns a well-known catalog of agents. Nested-catalog
// entries point at filtered GET /v1/catalog documents rather than listing
// every record inline.
func (c *aiFinderController) GetWellKnownCatalog(ctx context.Context, _ *catalogv1.GetWellKnownCatalogRequest) (*catalogv1.GetWellKnownCatalogResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	return &catalogv1.GetWellKnownCatalogResponse{
		Catalog: &catalogv1.AICatalog{
			SpecVersion: wellKnownSpecVersion,
			Host:        c.catalogHost(),
			Entries: []*catalogv1.CatalogEntry{
				c.nestedCatalogEntry("a2a", "A2A Agents",
					"Agents that publish an A2A agent card.",
					catalogv1.ProtocolA2ACardJsonMediaType, "a2a"),
				c.nestedCatalogEntry("mcp", "MCP Servers",
					"Agents that expose an MCP server connection.",
					catalogv1.ProtocolMCPCardJsonMediaType, "mcp"),
				c.nestedCatalogEntry("skill", "Agent Skills",
					"Agents that publish a reusable Agent Skill definition as SKILL.md.",
					catalogv1.ProtocolAgentSkillsMdMediaType, "skills"),
				c.nestedCatalogEntry("skill-bundle", "Agent Skill Bundles",
					"Agents that publish an Agent Skill directory bundled as .gzip.",
					catalogv1.ProtocolAgentSkillsBundleMediaType, "skills"),
			},
		},
	}, nil
}

// GetCatalog returns an AI Catalog document for one page of entries matching
// the same filter, order, and paging arguments as ListAgents.
func (c *aiFinderController) GetCatalog(ctx context.Context, req *catalogv1.GetCatalogRequest) (*catalogv1.GetCatalogResponse, error) {
	if req == nil {
		req = &catalogv1.GetCatalogRequest{}
	}

	aiFinderLogger.Debug("GetCatalog called", "filter", req.GetFilter(), "order_by", req.GetOrderBy(), "page_size", req.GetPageSize())

	entries, nextPageToken, _, err := c.listCatalogPage(ctx, req.GetFilter(), req.GetOrderBy(), req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, err
	}

	if entries == nil {
		entries = []*catalogv1.CatalogEntry{}
	}

	catalog := &catalogv1.AICatalog{
		SpecVersion: wellKnownSpecVersion,
		Host:        c.catalogHost(),
		Entries:     entries,
		Extensions:  map[string]*structpb.Value{},
	}

	if nextPageToken != "" {
		catalog.Extensions[catalogv1.CatalogPagingExtensionKey] = structpb.NewStructValue(&structpb.Struct{
			Fields: map[string]*structpb.Value{
				"nextPageToken": structpb.NewStringValue(nextPageToken),
			},
		})
	}

	return &catalogv1.GetCatalogResponse{Catalog: catalog}, nil
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

	entries, _, err := c.db.GetCatalogEntries(
		types.WithCIDs(cid),
		types.WithLimit(1),
	)
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

	// A record the policy gate excludes fails with the status that says so.
	var record *corev1.Record

	err = c.gate.Check(ctx, cid)
	if err == nil {
		record, err = c.store.Pull(ctx, &corev1.RecordRef{Cid: cid})
	}

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

// catalogHost identifies the directory operator on served AI Catalog documents.
func (c *aiFinderController) catalogHost() *catalogv1.HostInfo {
	identifier := c.cfg.PublicURL
	if identifier == "" {
		identifier = catalogv1.GetCatalogUrnFor("host", c.hostId)
	}

	return &catalogv1.HostInfo{
		DisplayName: wellKnownHostDisplayName,
		Identifier:  &identifier,
	}
}

// nestedCatalogEntry is a well-known pointer at a filtered GET /v1/catalog document.
func (c *aiFinderController) nestedCatalogEntry(name, displayName, description, mediaType, tag string) *catalogv1.CatalogEntry {
	return &catalogv1.CatalogEntry{
		Identifier:  catalogv1.GetCatalogUrnFor("catalog", name),
		DisplayName: displayName,
		Type:        catalogv1.CatalogMediaType,
		Description: new(description),
		Tags:        []string{tag},
		Artifact: &catalogv1.CatalogEntry_Url{
			Url: c.collectionURL(mediaType),
		},
	}
}

// collectionURL builds an absolute URL pointing at GET /v1/catalog
// constrained to a single AI Catalog type. The filter value is
// percent-encoded so the URL is safe to embed verbatim in JSON.
func (c *aiFinderController) collectionURL(mediaType string) string {
	base := strings.TrimRight(c.cfg.PublicURL, "/")

	v := url.Values{}
	v.Set("filter", "type="+mediaType)

	return fmt.Sprintf("%s/v1/catalog?%s", base, v.Encode())
}

// listCatalogPage applies the AI Finder browsing grammar and returns one page
// of catalog entries. A type= filter that matches no indexed module yields an
// empty page rather than an error.
func (c *aiFinderController) listCatalogPage(ctx context.Context, filter, orderBy string, pageSize uint32, pageToken string) ([]*catalogv1.CatalogEntry, string, uint32, error) {
	if err := ctx.Err(); err != nil {
		return nil, "", 0, status.Errorf(codes.Canceled, "%v", err)
	}

	parsedFilter, err := parseAgentFilter(filter)
	if err != nil {
		return nil, "", 0, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
	}

	// publisherId is part of the grammar but not yet indexed.
	if len(parsedFilter.PublisherIDs) > 0 {
		return nil, "", 0, status.Error(codes.Unimplemented, "publisherId filter is not yet supported") //nolint:wrapcheck
	}

	order, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, "", 0, status.Errorf(codes.InvalidArgument, "invalid order_by: %v", err)
	}

	offset, err := decodePageToken(pageToken)
	if err != nil {
		return nil, "", 0, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
	}

	limit := int(clampPageSize(pageSize))

	opts, ok := buildCatalogFilterOptions(parsedFilter, order, limit, offset)
	if !ok {
		return nil, "", 0, nil
	}

	totalCount, err := c.db.CountCatalogEntries(opts...)
	if err != nil {
		aiFinderLogger.Error("failed to count catalog entries", "error", err)

		return nil, "", 0, status.Error(codes.Internal, "failed to count catalog entries") //nolint:wrapcheck
	}

	entries, hasMore, err := c.db.GetCatalogEntries(opts...)
	if err != nil {
		aiFinderLogger.Error("failed to list catalog entries", "error", err)

		return nil, "", 0, status.Error(codes.Internal, "failed to list catalog entries") //nolint:wrapcheck
	}

	var nextPageToken string
	if hasMore {
		// Advance by the page size: the number of records consumed,
		// regardless of how many projected to an entry.
		nextPageToken = encodePageToken(offset + limit)
	}

	return entries, nextPageToken, totalCount, nil
}
