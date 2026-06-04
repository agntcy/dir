// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
	ctrl := NewAIFinderController(db)

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
	ctrl := NewAIFinderController(db)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{PageSize: 5})
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetNextPageToken())

	offset, err := decodePageToken(resp.GetNextPageToken())
	require.NoError(t, err)
	assert.Equal(t, 5, offset, "next offset advances by the page size")
}

func TestListAgents_PageTokenAppliesOffset(t *testing.T) {
	db := &fakeCatalogDB{}
	ctrl := NewAIFinderController(db)

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
	ctrl := NewAIFinderController(db)

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
	ctrl := NewAIFinderController(db)

	resp, err := ctrl.ListAgents(context.Background(), &catalogv1.ListAgentsRequest{Filter: "type=application/unknown+json"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Zero(t, db.calls, "unknown type short-circuits without hitting the DB")
}

func TestListAgents_Errors(t *testing.T) {
	ctrl := NewAIFinderController(&fakeCatalogDB{})

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
	ctrl := NewAIFinderController(db)

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
