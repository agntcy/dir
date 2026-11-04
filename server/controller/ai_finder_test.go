// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"sync"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/policy"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// errNotImplemented is returned by fakeStoreAPI stub methods that are not
// exercised by these tests.
var errNotImplemented = errors.New("not implemented")

// fakeCatalogDB implements the AI Finder database surface, capturing the
// filters it was called with and returning canned results.
type fakeCatalogDB struct {
	entries []*catalogv1.CatalogEntry
	tags    []*catalogv1.CatalogTag
	err     error
	// listErr fails only GetCatalogEntries, leaving the preceding
	// CountCatalogEntries call to succeed.
	listErr error
	calls   int
	// recordCIDs is returned by GetRecordCIDs, keyed by the query value the
	// search fan-out asked for. The "" key answers any unmatched query.
	recordCIDs map[string][]string
	// excluded makes the gate report every record as excluded by
	// the policy gate; the zero value serves everything.
	excluded   bool
	gotFilters types.CatalogFilters

	mu              sync.Mutex
	gotRecordQuery  []types.RecordFilters
	recordCallCount int
}

// GetRecordCIDs answers one fan-out signal query. The fan-out runs its signals
// concurrently, so this records calls under a mutex.
func (f *fakeCatalogDB) Check(_ context.Context, cid string) error {
	return gateAnswer(cid, !f.excluded, nil)
}

// servesEverything is a gate with no verdict enforced and no policy at pull,
// which is a node that serves whatever it stores.
func servesEverything() *policy.RecordGate { return policy.NewRecordGate(nil, nil) }

func (f *fakeCatalogDB) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	cfg := types.RecordFilters{}

	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	f.mu.Lock()
	f.gotRecordQuery = append(f.gotRecordQuery, cfg)
	f.recordCallCount++
	f.mu.Unlock()

	if f.err != nil {
		return nil, f.err
	}

	for _, values := range [][]string{cfg.SkillNames, cfg.DomainNames, cfg.Names, cfg.Descriptions} {
		for _, v := range values {
			if cids, ok := f.recordCIDs[v]; ok {
				return cids, nil
			}
		}
	}

	return f.recordCIDs[""], nil
}

func (f *fakeCatalogDB) captureFilters(opts ...types.CatalogQueryOption) {
	cfg := types.CatalogFilters{}

	for _, opt := range opts {
		if opt != nil {
			opt.ApplyCatalog(&cfg)
		}
	}

	f.gotFilters = cfg
}

func (f *fakeCatalogDB) GetCatalogEntries(opts ...types.CatalogQueryOption) ([]*catalogv1.CatalogEntry, bool, error) {
	f.calls++
	f.captureFilters(opts...)

	if f.err != nil {
		return nil, false, f.err
	}

	if f.listErr != nil {
		return nil, false, f.listErr
	}

	entries := f.entries

	// Honor WithCIDs so hydration tests (and GetAgent) see only the requested
	// records, mirroring the real catalog query.
	if len(f.gotFilters.CIDs) > 0 {
		want := make(map[string]struct{}, len(f.gotFilters.CIDs))
		for _, cid := range f.gotFilters.CIDs {
			want[cid] = struct{}{}
		}

		filtered := entries[:0:0]

		for _, e := range entries {
			if _, ok := want[catalogEntryCID(e)]; ok {
				filtered = append(filtered, e)
			}
		}

		entries = filtered
	}

	if f.gotFilters.Offset > 0 {
		if f.gotFilters.Offset >= len(entries) {
			return nil, false, nil
		}

		entries = entries[f.gotFilters.Offset:]
	}

	limit := f.gotFilters.Limit
	if limit <= 0 {
		limit = 20
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	return entries, hasMore, nil
}

func (f *fakeCatalogDB) CountCatalogEntries(opts ...types.CatalogQueryOption) (uint32, error) {
	f.captureFilters(opts...)

	if f.err != nil {
		return 0, f.err
	}

	return uint32(len(f.entries)), nil //nolint:gosec
}

func (f *fakeCatalogDB) ListCatalogTags() ([]*catalogv1.CatalogTag, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.tags, nil
}

func entry(id string) *catalogv1.CatalogEntry {
	return &catalogv1.CatalogEntry{Identifier: id, DisplayName: id, Type: "application/a2a-agent-card+json"}
}

func TestListAgents_DefaultsAndResults(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a"), entry("b")}}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.GetResults(), 2)
	assert.Equal(t, uint32(2), resp.GetTotalCount())
	assert.Empty(t, resp.GetNextPageToken(), "no more results means no token")

	// Defaults: page size 20, newest-first ordering.
	assert.Equal(t, 20, db.gotFilters.Limit)
	assert.Equal(t, 0, db.gotFilters.Offset)
	require.Len(t, db.gotFilters.OrderBy, 1)
	assert.Equal(t, types.RecordOrderClause{Column: "created_at", Desc: true}, db.gotFilters.OrderBy[0])
}

func TestListAgents_NextPageToken(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a"), entry("b"), entry("c"), entry("d"), entry("e"), entry("f")}}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{PageSize: 5})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetNextPageToken())
	assert.Equal(t, uint32(6), resp.GetTotalCount())
	assert.Len(t, resp.GetResults(), 5)

	offset, err := decodePageToken(resp.GetNextPageToken())
	require.NoError(t, err)
	assert.Equal(t, 5, offset, "next offset advances by the page size")
}

func TestListAgents_PageTokenAppliesOffset(t *testing.T) {
	db := &fakeCatalogDB{}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{
		PageSize:  10,
		PageToken: encodePageToken(30),
	})
	require.NoError(t, err)
	assert.Equal(t, 30, db.gotFilters.Offset)
	assert.Equal(t, 10, db.gotFilters.Limit)
}

func TestListAgents_FilterTranslation(t *testing.T) {
	db := &fakeCatalogDB{}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{
		Filter: `displayName=weather AND type=application/a2a-agent-card+json AND createdAfter=2024-01-01T00:00:00Z`,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"*weather*"}, db.gotFilters.Names)
	require.Len(t, db.gotFilters.MediaTypeFilters, 1)
	assert.Equal(t, "integration/a2a", db.gotFilters.MediaTypeFilters[0].ModuleName)
	assert.Empty(t, db.gotFilters.MediaTypeFilters[0].ArtifactMediaType)
	assert.Equal(t, []string{">2024-01-01T00:00:00Z"}, db.gotFilters.CreatedAts)
}

func TestListAgents_UnknownTypeYieldsZeroRows(t *testing.T) {
	db := &fakeCatalogDB{}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{Filter: "type=application/unknown+json"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Zero(t, resp.GetTotalCount())
	assert.Zero(t, db.calls, "unknown type short-circuits without hitting the DB")
}

func TestListAgents_Errors(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	tests := []struct {
		name string
		req  *catalogv1.ListAgentsRequest
		code codes.Code
	}{
		{"bad filter", &catalogv1.ListAgentsRequest{Filter: "displayName"}, codes.InvalidArgument},
		{"bad order_by", &catalogv1.ListAgentsRequest{OrderBy: "bogus"}, codes.InvalidArgument},
		{"bad page_token", &catalogv1.ListAgentsRequest{PageToken: "!!!not-base64!!!"}, codes.InvalidArgument},
		{"publisherId unsupported", &catalogv1.ListAgentsRequest{Filter: "publisherId=acme"}, codes.Unimplemented},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ctrl.ListAgents(context.Background(), tc.req)
			require.Error(t, err)
			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}

func TestListAgents_DBError(t *testing.T) {
	db := &fakeCatalogDB{err: assert.AnError}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestParseAgentFilter(t *testing.T) {
	t.Run("empty is a no-op", func(t *testing.T) {
		f, err := parseAgentFilter("")
		require.NoError(t, err)
		assert.Equal(t, agentFilter{}, f)
	})

	t.Run("AND across fields, comma-OR within", func(t *testing.T) {
		f, err := parseAgentFilter(`type=a,b AND displayName=foo`)
		require.NoError(t, err)
		assert.Equal(t, "foo", f.DisplayName)
		assert.Equal(t, []string{"a", "b"}, f.Types)
	})

	t.Run("quoted value keeps commas and spaces", func(t *testing.T) {
		f, err := parseAgentFilter(`displayName="hello, world"`)
		require.NoError(t, err)
		assert.Equal(t, "hello, world", f.DisplayName)
	})

	t.Run("verified=true", func(t *testing.T) {
		f, err := parseAgentFilter(`verified=true`)
		require.NoError(t, err)
		require.NotNil(t, f.Verified)
		assert.True(t, *f.Verified)
	})

	t.Run("verified=false", func(t *testing.T) {
		f, err := parseAgentFilter(`verified=FALSE`)
		require.NoError(t, err)
		require.NotNil(t, f.Verified)
		assert.False(t, *f.Verified)
	})

	t.Run("trusted=true", func(t *testing.T) {
		f, err := parseAgentFilter(`trusted=true`)
		require.NoError(t, err)
		require.NotNil(t, f.Trusted)
		assert.True(t, *f.Trusted)
	})

	t.Run("safe=true", func(t *testing.T) {
		f, err := parseAgentFilter(`safe=true`)
		require.NoError(t, err)
		require.NotNil(t, f.Safe)
		assert.True(t, *f.Safe)
	})

	t.Run("tags skill tag", func(t *testing.T) {
		f, err := parseAgentFilter(`tags=oasf:*:skills:natural_language_processing/*`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{{SkillName: "natural_language_processing/*"}}, f.TagFilters)
	})

	t.Run("tags comma-OR", func(t *testing.T) {
		f, err := parseAgentFilter(`tags=oasf:*:skills:foo/*,oasf:*:skills:bar`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{
			{SkillName: "foo/*"},
			{SkillName: "bar"},
		}, f.TagFilters)
	})

	t.Run("tags domain tag", func(t *testing.T) {
		f, err := parseAgentFilter(`tags=oasf:*:domains:healthcare/*`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{{DomainName: "healthcare/*"}}, f.TagFilters)
	})

	t.Run("tags mixed skill and domain", func(t *testing.T) {
		f, err := parseAgentFilter(`tags=oasf:*:skills:foo,oasf:*:domains:bar`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{
			{SkillName: "foo"},
			{DomainName: "bar"},
		}, f.TagFilters)
	})

	t.Run("tags annotation pair", func(t *testing.T) {
		f, err := parseAgentFilter(`tags="owner=alice"`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{{Annotation: &types.Annotation{Key: "owner", Value: "alice"}}}, f.TagFilters)
	})

	t.Run("tags annotation key only", func(t *testing.T) {
		f, err := parseAgentFilter(`tags=owner`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{{AnnotationKey: "owner"}}, f.TagFilters)
	})

	t.Run("tags mixed catalog tag kinds", func(t *testing.T) {
		f, err := parseAgentFilter(`tags=oasf:*:skills:foo,"owner=alice",env`)
		require.NoError(t, err)
		assert.Equal(t, []types.TagFilter{
			{SkillName: "foo"},
			{Annotation: &types.Annotation{Key: "owner", Value: "alice"}},
			{AnnotationKey: "env"},
		}, f.TagFilters)
	})

	invalid := []struct {
		name  string
		input string
	}{
		{"no equals", "displayName"},
		{"unknown field", "color=red"},
		{"duplicate field", "displayName=a AND displayName=b"},
		{"empty value", "displayName="},
		{"multi-value displayName", "displayName=a,b"},
		{"bad timestamp", "createdAfter=not-a-date"},
		{"bad verified value", "verified=maybe"},
		{"multi-value verified", "verified=true,false"},
		{"bad trusted value", "trusted=maybe"},
		{"bad safe value", "safe=maybe"},
		{"empty annotation key", "tags==alice"},
		{"unterminated quote", `displayName="oops`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseAgentFilter(tc.input)
			require.Error(t, err)
		})
	}
}

func TestBuildCatalogFilterOptionsVerified(t *testing.T) {
	verified := true
	f := agentFilter{Verified: &verified}

	opts, ok := buildCatalogFilterOptions(f, nil, 20, 0)
	require.True(t, ok)

	cfg := types.CatalogFilters{}
	for _, opt := range opts {
		opt.ApplyCatalog(&cfg)
	}

	require.NotNil(t, cfg.Verified)
	assert.True(t, *cfg.Verified)
}

func TestBuildCatalogFilterOptionsTrusted(t *testing.T) {
	trusted := true
	f := agentFilter{Trusted: &trusted}

	opts, ok := buildCatalogFilterOptions(f, nil, 20, 0)
	require.True(t, ok)

	cfg := types.CatalogFilters{}
	for _, opt := range opts {
		opt.ApplyCatalog(&cfg)
	}

	require.NotNil(t, cfg.Trusted)
	assert.True(t, *cfg.Trusted)
}

func TestBuildCatalogFilterOptionsSafe(t *testing.T) {
	safe := true
	f := agentFilter{Safe: &safe}

	opts, ok := buildCatalogFilterOptions(f, nil, 20, 0)
	require.True(t, ok)

	cfg := types.CatalogFilters{}
	for _, opt := range opts {
		opt.ApplyCatalog(&cfg)
	}

	require.NotNil(t, cfg.ScanSafe)
	assert.True(t, *cfg.ScanSafe)
}

func TestBuildCatalogFilterOptionsTags(t *testing.T) {
	f := agentFilter{
		TagFilters: []types.TagFilter{
			{SkillName: "natural_language_processing/*"},
			{SkillName: "coding/*"},
			{DomainName: "healthcare/*"},
			{Annotation: &types.Annotation{Key: "owner", Value: "alice"}},
			{AnnotationKey: "env"},
		},
	}

	opts, ok := buildCatalogFilterOptions(f, nil, 20, 0)
	require.True(t, ok)

	cfg := types.CatalogFilters{}
	for _, opt := range opts {
		opt.ApplyCatalog(&cfg)
	}

	assert.Equal(t, []types.TagFilter{
		{SkillName: "natural_language_processing/*"},
		{SkillName: "coding/*"},
		{DomainName: "healthcare/*"},
		{Annotation: &types.Annotation{Key: "owner", Value: "alice"}},
		{AnnotationKey: "env"},
	}, cfg.TagFilters)
}

func TestTagFilterForTag(t *testing.T) {
	filter := tagFilterForTag("oasf:*:skills:retrieval_augmented_generation/*")
	require.NotNil(t, filter)
	assert.Equal(t, "retrieval_augmented_generation/*", filter.SkillName)

	filter = tagFilterForTag("oasf:*:domains:healthcare/*")
	require.NotNil(t, filter)
	assert.Equal(t, "healthcare/*", filter.DomainName)

	filter = tagFilterForTag("owner=alice")
	require.NotNil(t, filter)
	assert.Equal(t, &types.Annotation{Key: "owner", Value: "alice"}, filter.Annotation)

	filter = tagFilterForTag("env")
	require.NotNil(t, filter)
	assert.Equal(t, "env", filter.AnnotationKey)
}

func TestParseOrderBy(t *testing.T) {
	t.Run("empty yields default", func(t *testing.T) {
		got, err := parseOrderBy("")
		require.NoError(t, err)
		assert.Equal(t, defaultAgentOrder(), got)
	})

	t.Run("fields with directions", func(t *testing.T) {
		got, err := parseOrderBy("display_name, created_at DESC")
		require.NoError(t, err)
		assert.Equal(t, []orderByClause{{Column: "name", Desc: false}, {Column: "created_at", Desc: true}}, got)
	})

	invalid := []string{"unknown_field", "name BADDIR", "name, name", "a,b,c,d,e,f,g,h,i"}
	for _, in := range invalid {
		t.Run(in, func(t *testing.T) {
			_, err := parseOrderBy(in)
			require.Error(t, err)
		})
	}
}

func TestPageTokenRoundTrip(t *testing.T) {
	assert.Empty(t, encodePageToken(0))

	tok := encodePageToken(42)
	assert.Equal(t, "42", tok)

	offset, err := decodePageToken(tok)
	require.NoError(t, err)
	assert.Equal(t, 42, offset)

	got, err := decodePageToken("")
	require.NoError(t, err)
	assert.Zero(t, got)

	_, err = decodePageToken("###")
	require.Error(t, err)
}

func TestClampPageSize(t *testing.T) {
	assert.Equal(t, uint32(20), clampPageSize(0))
	assert.Equal(t, uint32(50), clampPageSize(50))
	assert.Equal(t, uint32(100), clampPageSize(1000))
}

func TestWellKnown(t *testing.T) {
	ctrl := NewAIFinderController("hostId-name", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{
		PublicURL: "https://ai-catalog.example",
	}, nil)

	resp, err := ctrl.GetWellKnownCatalog(t.Context(), nil)
	require.NoError(t, err)

	catalog := resp.GetCatalog()
	require.NotNil(t, catalog)
	assert.Equal(t, wellKnownSpecVersion, catalog.GetSpecVersion())
	assert.Equal(t, wellKnownHostDisplayName, catalog.GetHost().GetDisplayName())
	assert.Equal(t, "https://ai-catalog.example", catalog.GetHost().GetIdentifier())
	require.Len(t, catalog.GetEntries(), 4)

	expected := map[string]struct{ filter, identifier string }{
		"A2A Agents":          {catalogv1.ProtocolA2ACardJsonMediaType, "urn:air:org.agntcy:catalog:a2a"},
		"MCP Servers":         {catalogv1.ProtocolMCPCardJsonMediaType, "urn:air:org.agntcy:catalog:mcp"},
		"Agent Skills":        {catalogv1.ProtocolAgentSkillsMdMediaType, "urn:air:org.agntcy:catalog:skill"},
		"Agent Skill Bundles": {catalogv1.ProtocolAgentSkillsBundleMediaType, "urn:air:org.agntcy:catalog:skill-bundle"},
	}

	for _, entry := range catalog.GetEntries() {
		want, ok := expected[entry.GetDisplayName()]
		require.True(t, ok, "unexpected nested catalog %q", entry.GetDisplayName())
		assert.Equal(t, catalogv1.CatalogMediaType, entry.GetType())
		assert.Equal(t, want.identifier, entry.GetIdentifier())

		nestedURL, err := url.Parse(entry.GetUrl())
		require.NoError(t, err)
		assert.Equal(t, "/v1/catalog", nestedURL.Path)
		assert.Equal(t, "type="+want.filter, nestedURL.Query().Get("filter"))
		delete(expected, entry.GetDisplayName())
	}

	assert.Empty(t, expected)
	assertAICatalogWire(t, catalog)
}

func TestGetCatalog(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a"), entry("b")}}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{
		PublicURL: "https://ai-catalog.example",
	}, nil)

	resp, err := ctrl.GetCatalog(context.Background(), &catalogv1.GetCatalogRequest{})
	require.NoError(t, err)

	catalog := resp.GetCatalog()
	require.NotNil(t, catalog)
	assert.Equal(t, wellKnownSpecVersion, catalog.GetSpecVersion())
	assert.Equal(t, wellKnownHostDisplayName, catalog.GetHost().GetDisplayName())
	require.Len(t, catalog.GetEntries(), 2)
	assert.Equal(t, "a", catalog.GetEntries()[0].GetIdentifier())
	_, hasPaging := catalog.GetExtensions()[catalogv1.CatalogPagingExtensionKey]
	assert.False(t, hasPaging)
	assertAICatalogWire(t, catalog)
}

// GetCatalog is reachable over gRPC with no request message, and over the
// gateway for a store that has nothing to publish yet; both must still yield
// a well-formed, empty catalog rather than a nil entries list.
func TestGetCatalog_NilRequestAndEmptyStore(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.GetCatalog(context.Background(), nil)
	require.NoError(t, err)

	catalog := resp.GetCatalog()
	require.NotNil(t, catalog)
	assert.Equal(t, wellKnownSpecVersion, catalog.GetSpecVersion())
	assert.NotNil(t, catalog.GetEntries())
	assert.Empty(t, catalog.GetEntries())
	assert.Empty(t, catalog.GetExtensions())

	// An empty catalog must still marshal with an entries field present.
	raw, err := protojson.Marshal(catalog)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "specVersion")
}

func TestGetCatalog_DBError(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{err: assert.AnError}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetCatalog(context.Background(), &catalogv1.GetCatalogRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestGetCatalog_ListError(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a")}, listErr: assert.AnError}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetCatalog(context.Background(), &catalogv1.GetCatalogRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestCatalogEndpoints_CanceledContext(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ctrl.GetWellKnownCatalog(ctx, nil)
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))

	_, err = ctrl.GetCatalog(ctx, &catalogv1.GetCatalogRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
}

func assertAICatalogWire(t *testing.T, catalog *catalogv1.AICatalog) {
	t.Helper()

	raw, err := protojson.Marshal(catalog)
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	assert.Equal(t, wellKnownSpecVersion, doc["specVersion"])
	assert.NotEmpty(t, doc["entries"])
	_, hasCollections := doc["collections"]
	assert.False(t, hasCollections)
}

func TestGetCatalog_PagingExtension(t *testing.T) {
	entries := []*catalogv1.CatalogEntry{entry("a"), entry("b"), entry("c")}
	db := &fakeCatalogDB{entries: entries}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.GetCatalog(context.Background(), &catalogv1.GetCatalogRequest{PageSize: 2})
	require.NoError(t, err)

	catalog := resp.GetCatalog()
	require.Len(t, catalog.GetEntries(), 2)
	paging, hasPaging := catalog.GetExtensions()[catalogv1.CatalogPagingExtensionKey]
	require.True(t, hasPaging)
	require.NotNil(t, paging)
	assert.NotEmpty(t, paging.GetStructValue().AsMap()["nextPageToken"])
	assertAICatalogWire(t, catalog)
}

// fakeStoreAPI implements types.StoreAPI for testing.
type fakeStoreAPI struct {
	record *corev1.Record
	err    error
}

func (f *fakeStoreAPI) Push(context.Context, *corev1.Record) (*corev1.RecordRef, error) {
	return nil, errNotImplemented
}

func (f *fakeStoreAPI) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.record, nil
}

func (f *fakeStoreAPI) Lookup(context.Context, *corev1.RecordRef) (*corev1.RecordMeta, error) {
	return nil, errNotImplemented
}

func (f *fakeStoreAPI) Delete(context.Context, *corev1.RecordRef) error {
	return nil
}

func (f *fakeStoreAPI) IsReady(context.Context) bool {
	return true
}

func TestGetAgent_Success(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("cid123")}}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: "cid123"})
	require.NoError(t, err)
	assert.Equal(t, "cid123", resp.GetEntry().GetIdentifier())
	assert.Equal(t, []string{"cid123"}, db.gotFilters.CIDs)
	assert.Equal(t, 1, db.gotFilters.Limit)
}

func TestGetAgent_NotFound(t *testing.T) {
	db := &fakeCatalogDB{entries: nil}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: "missing"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetAgent_EmptyCID(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetAgent_NilRequest(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetAgent_DBError(t *testing.T) {
	db := &fakeCatalogDB{err: assert.AnError}
	ctrl := NewAIFinderController("hostId", db, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: "cid123"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestExportAgent_Success(t *testing.T) {
	record := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.5.0",
		Description:   "A test agent",
		Version:       "1.0.0",
	})
	store := &fakeStoreAPI{record: record}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, store)

	resp, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123"})
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.GetContentType())
	assert.Contains(t, string(resp.GetData()), "test-agent")
}

func TestExportAgent_DefaultFormat(t *testing.T) {
	record := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "v0.5.0",
	})
	store := &fakeStoreAPI{record: record}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, store)

	resp, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123", Format: ""})
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.GetContentType())
}

func TestExportAgent_NoStore(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123"})
	require.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestExportAgent_EmptyCID(t *testing.T) {
	store := &fakeStoreAPI{}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExportAgent_NilRequest(t *testing.T) {
	store := &fakeStoreAPI{}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExportAgent_StoreNotFound(t *testing.T) {
	store := &fakeStoreAPI{err: status.Error(codes.NotFound, "not found")}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "missing"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestExportAgent_NilData(t *testing.T) {
	store := &fakeStoreAPI{record: &corev1.Record{}}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, servesEverything(), config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestBuildTypeFilterOptions(t *testing.T) {
	t.Run("agent skills markdown", func(t *testing.T) {
		filters, ok := buildTypeFilterOptions([]string{catalogv1.ProtocolAgentSkillsMdMediaType})
		require.True(t, ok)
		require.Len(t, filters, 1)
		assert.Equal(t, translator.AgentSkillsModuleName, filters[0].ModuleName)
		assert.Equal(t, catalogv1.ProtocolAgentSkillsMdMediaType, filters[0].ArtifactMediaType)
	})

	t.Run("agent skills bundle", func(t *testing.T) {
		filters, ok := buildTypeFilterOptions([]string{catalogv1.ProtocolAgentSkillsBundleMediaType})
		require.True(t, ok)
		require.Len(t, filters, 1)
		assert.Equal(t, catalogv1.ProtocolAgentSkillsBundleMediaType, filters[0].ArtifactMediaType)
	})

	t.Run("unknown type", func(t *testing.T) {
		_, ok := buildTypeFilterOptions([]string{"application/unknown"})
		assert.False(t, ok)
	})
}

func TestListTags_ReturnsSortedTags(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{
		tags: []*catalogv1.CatalogTag{
			{
				Id:    catalogv1.SkillTag("*", "test_skill"),
				Label: "Test Skill",
			},
			{
				Id:    catalogv1.DomainTag("*", "life_science/biotechnology"),
				Label: "Biotechnology",
			},
		},
	}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListTags(context.Background(), &catalogv1.ListTagsRequest{})
	require.NoError(t, err)
	require.Len(t, resp.GetTags(), 2)
	assert.Equal(t, catalogv1.SkillTag("*", "test_skill"), resp.GetTags()[0].GetId())
	assert.Equal(t, "Test Skill", resp.GetTags()[0].GetLabel())
	assert.Equal(t, catalogv1.DomainTag("*", "life_science/biotechnology"), resp.GetTags()[1].GetId())
	assert.Equal(t, "Biotechnology", resp.GetTags()[1].GetLabel())
}

func TestListTags_DatabaseError(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{
		err: errors.New("db down"),
	}, servesEverything(), config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ListTags(context.Background(), &catalogv1.ListTagsRequest{})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}
