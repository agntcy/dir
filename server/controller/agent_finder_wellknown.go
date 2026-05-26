// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AI Catalog Well-Known endpoint — see RFC 8615 + the AI Catalog spec.
//
// /.well-known/ai-catalog.json publishes:
//
//   - the host descriptor (identity + display name), and
//   - a static set of collections — pre-built /v1/agents queries — so
//     consumers can discover the dynamic catalog without having to read
//     this server's filter grammar.
//
// The `entries` array (records explicitly promoted to the well-known
// surface) is intentionally empty for now. See the TODO in
// GetWellKnownCatalog below.

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

	// wellKnownHostIdentifier matches the URN authority emitted by the
	// catalog entry projection (catalogURNHost in the data layer).
	wellKnownHostIdentifier = "agntcy.org"

	// wellKnownFallbackBaseURL is used when the controller is constructed
	// without an explicit base URL (e.g. tests, direct gRPC clients that
	// don't go through the HTTP gateway). It mirrors the default gateway
	// listen address (config.DefaultHTTPGatewayAddress, port-only :8889)
	// resolved to a clientable host.
	wellKnownFallbackBaseURL = "http://localhost:8889"
)

// GetWellKnownCatalog implements GET /.well-known/ai-catalog.json.
//
// The payload is assembled from two sources:
//
//  1. Static host descriptor (hardcoded today; see TODO above).
//  2. Static collections — one per supported AI Catalog media type —
//     anchored on the controller's publicBaseURL.
//
// The endpoint is parameter-less by spec; the request message exists
// only for forward compatibility.
//
// TODO(ai-catalog): populate `entries` from records explicitly promoted
// to the well-known surface. This requires (a) plumbing a "published"
// signal through the record schema and the RecordFilters surface, and
// (b) a write path that flips records on/off the well-known list. Until
// that lands, the well-known document is a pure discovery surface
// (host + collections) and `entries` is always empty — which is
// spec-compliant for Level 1 (Minimal) hosts.
func (c *agentFinderCtlr) GetWellKnownCatalog(
	ctx context.Context,
	_ *catalogv1.GetWellKnownCatalogRequest,
) (*catalogv1.GetWellKnownCatalogResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, status.Errorf(codes.Canceled, "%v", err)
	}

	hostID := wellKnownHostIdentifier

	return &catalogv1.GetWellKnownCatalogResponse{
		Catalog: &catalogv1.WellKnownCatalog{
			SpecVersion: wellKnownSpecVersion,
			Host: &catalogv1.HostInfo{
				DisplayName: wellKnownHostDisplayName,
				Identifier:  &hostID,
			},
			Entries:     nil, // TODO: published entries — see function doc.
			Collections: c.defaultCollections(),
		},
	}, nil
}

// defaultCollections returns the static set of CatalogCollections
// published in the well-known document. Order is deterministic so
// consumers can diff the payload across calls.
func (c *agentFinderCtlr) defaultCollections() []*catalogv1.CatalogCollection {
	descriptors := []struct {
		displayName string
		description string
		mediaType   string // OASF-side AI Catalog media type
	}{
		{
			displayName: "A2A Agents",
			description: "Agents that publish an A2A agent card.",
			mediaType:   "application/a2a-agent-card+json",
		},
		{
			displayName: "MCP Servers",
			description: "Agents that expose an MCP server connection.",
			mediaType:   "application/mcp-server+json",
		},
		{
			displayName: "AI Skills",
			description: "Agents that publish a reusable AI skill definition.",
			mediaType:   "application/ai-skill+md",
		},
	}

	out := make([]*catalogv1.CatalogCollection, 0, len(descriptors))

	for _, d := range descriptors {
		desc := d.description
		mt := types.CatalogContainerMediaType

		out = append(out, &catalogv1.CatalogCollection{
			DisplayName: d.displayName,
			Url:         c.collectionURL(d.mediaType),
			Description: &desc,
			MediaType:   &mt,
		})
	}

	return out
}

// collectionURL builds an absolute URL pointing at GET /v1/agents
// constrained to a single AI Catalog media type. The filter value is
// percent-encoded so the URL is safe to embed verbatim in JSON.
func (c *agentFinderCtlr) collectionURL(mediaType string) string {
	base := strings.TrimRight(c.publicBaseURL, "/")
	if base == "" {
		base = wellKnownFallbackBaseURL
	}

	v := url.Values{}
	v.Set("filter", "type="+mediaType)

	return fmt.Sprintf("%s/v1/agents?%s", base, v.Encode())
}
