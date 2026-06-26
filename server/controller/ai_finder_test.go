// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errNotImplemented is returned by fakeStoreAPI stub methods that are not
// exercised by these tests.
var errNotImplemented = errors.New("not implemented")

// fakeCatalogDB implements types.CatalogDatabaseAPI, capturing the filters it
// was called with and returning canned results.
type fakeCatalogDB struct {
	entries    []*catalogv1.CatalogEntry
	hasMore    bool
	err        error
	calls      int
	gotFilters types.RecordFilters
}

func (f *fakeCatalogDB) GetCatalogEntries(opts ...types.FilterOption) ([]*catalogv1.CatalogEntry, bool, error) {
	f.calls++

	cfg := types.RecordFilters{}
	for _, o := range opts {
		o(&cfg)
	}

	f.gotFilters = cfg

	return f.entries, f.hasMore, f.err
}

func entry(id string) *catalogv1.CatalogEntry {
	return &catalogv1.CatalogEntry{Identifier: id, DisplayName: id, MediaType: "application/a2a-agent-card+json"}
}

func TestListAgents_DefaultsAndResults(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a"), entry("b")}}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{})
	require.NoError(t, err)
	assert.Len(t, resp.GetResults(), 2)
	assert.Empty(t, resp.GetNextPageToken(), "no more results means no token")

	// Defaults: page size 20, newest-first ordering.
	assert.Equal(t, 20, db.gotFilters.Limit)
	assert.Equal(t, 0, db.gotFilters.Offset)
	require.Len(t, db.gotFilters.OrderBy, 1)
	assert.Equal(t, types.RecordOrderClause{Column: "created_at", Desc: true}, db.gotFilters.OrderBy[0])
}

func TestListAgents_NextPageToken(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a")}, hasMore: true}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{PageSize: 5})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetNextPageToken())

	offset, err := decodePageToken(resp.GetNextPageToken())
	require.NoError(t, err)
	assert.Equal(t, 5, offset, "next offset advances by the page size")
}

func TestListAgents_PageTokenAppliesOffset(t *testing.T) {
	db := &fakeCatalogDB{}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

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
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{
		Filter: `displayName=weather AND type=application/a2a-agent-card+json AND createdAfter=2024-01-01T00:00:00Z`,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"*weather*"}, db.gotFilters.Names)
	assert.Equal(t, []string{"integration/a2a"}, db.gotFilters.ModuleNames)
	assert.Equal(t, []string{">2024-01-01T00:00:00Z"}, db.gotFilters.CreatedAts)
}

func TestListAgents_UnknownTypeYieldsZeroRows(t *testing.T) {
	db := &fakeCatalogDB{}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{Filter: "type=application/unknown+json"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Zero(t, db.calls, "unknown type short-circuits without hitting the DB")
}

func TestListAgents_Errors(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

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
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

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
		{"unterminated quote", `displayName="oops`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseAgentFilter(tc.input)
			require.Error(t, err)
		})
	}
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
	require.NotEmpty(t, tok)

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
	ctrl := NewAIFinderController("hostId-name", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.GetWellKnownCatalog(t.Context(), nil)
	require.NoError(t, err)

	assert.Contains(t, resp.GetCatalog().GetHost().GetTrustManifest().GetIdentity(), "hostId-name")
	assert.Len(t, resp.GetCatalog().GetCollections(), 3)
	assert.Empty(t, resp.GetCatalog().GetEntries())
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
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

	resp, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: "cid123"})
	require.NoError(t, err)
	assert.Equal(t, "cid123", resp.GetEntry().GetIdentifier())
	assert.Equal(t, []string{"cid123"}, db.gotFilters.CIDs)
	assert.Equal(t, 1, db.gotFilters.Limit)
}

func TestGetAgent_NotFound(t *testing.T) {
	db := &fakeCatalogDB{entries: nil}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: "missing"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestGetAgent_EmptyCID(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), &catalogv1.GetAgentRequest{Cid: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetAgent_NilRequest(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.GetAgent(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestGetAgent_DBError(t *testing.T) {
	db := &fakeCatalogDB{err: assert.AnError}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil)

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
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, store)

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
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, store)

	resp, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123", Format: ""})
	require.NoError(t, err)
	assert.Equal(t, "application/json", resp.GetContentType())
}

func TestExportAgent_NoStore(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123"})
	require.Error(t, err)
	assert.Equal(t, codes.Unimplemented, status.Code(err))
}

func TestExportAgent_EmptyCID(t *testing.T) {
	store := &fakeStoreAPI{}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: ""})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExportAgent_NilRequest(t *testing.T) {
	store := &fakeStoreAPI{}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), nil)
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExportAgent_StoreNotFound(t *testing.T) {
	store := &fakeStoreAPI{err: status.Error(codes.NotFound, "not found")}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "missing"})
	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestExportAgent_NilData(t *testing.T) {
	store := &fakeStoreAPI{record: &corev1.Record{}}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, store)

	_, err := ctrl.ExportAgent(context.Background(), &catalogv1.ExportAgentRequest{Cid: "cid123"})
	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestOASFModuleForMediaType_AgentSkillBundle(t *testing.T) {
	module, ok := oasfModuleForMediaType(catalogv1.ProtocolAgentSkillsBundleMediaType)
	assert.True(t, ok)
	assert.Equal(t, "core/language_model/agentskills", module)
}

func TestFilterCatalogEntriesByMediaType(t *testing.T) {
	entries := []*catalogv1.CatalogEntry{
		{MediaType: catalogv1.ProtocolAgentSkillsMdMediaType},
		{MediaType: catalogv1.ProtocolAgentSkillsBundleMediaType},
		{MediaType: catalogv1.ProtocolMCPCardJsonMediaType},
	}

	got := filterCatalogEntriesByMediaType(entries, []string{
		catalogv1.ProtocolAgentSkillsBundleMediaType,
	})
	require.Len(t, got, 1)
	assert.Equal(t, catalogv1.ProtocolAgentSkillsBundleMediaType, got[0].GetMediaType())
}
