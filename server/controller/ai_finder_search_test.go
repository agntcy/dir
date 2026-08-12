// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/config"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeExtractor is defined alongside the ExtractTaxonomy tests; these tests
// reuse it rather than declaring a second one in the same package.

// resultWith builds an extractor Result from skill and domain names, all in the
// closest tier so Decompose keeps them.
func resultWith(skills []string, domains []string) sdk.Result {
	r := sdk.Result{}
	for _, s := range skills {
		r.Skills = append(r.Skills, sdk.ScoredClass{Class: sdk.Class{Name: s}, Score: 0.9, Tier: 1})
	}

	for _, d := range domains {
		r.Domains = append(r.Domains, sdk.ScoredClass{Class: sdk.Class{Name: d}, Score: 0.9, Tier: 1})
	}

	return r
}

func entryIDs(entries []*catalogv1.CatalogEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.GetIdentifier())
	}

	return ids
}

func searchCtrl(db *fakeCatalogDB, ext *fakeExtractor) catalogv1.AIFinderServiceServer {
	return NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))
}

func TestSearchAgents_RanksBySignalOverlap(t *testing.T) {
	// "b" matches both the skill and the domain, so it outranks records that
	// matched only one — the union is ranked, not filtered down to "b".
	db := &fakeCatalogDB{
		entries: []*catalogv1.CatalogEntry{entry("a"), entry("b"), entry("c")},
		recordCIDs: map[string][]string{
			"code_review":  {"a", "b"},
			"software_dev": {"b", "c"},
		},
	}
	ext := &fakeExtractor{result: resultWith([]string{"code_review"}, []string{"software_dev"})}

	resp, err := searchCtrl(db, ext).SearchAgents(context.Background(),
		&catalogv1.SearchAgentsRequest{Query: "review my code"})
	require.NoError(t, err)

	assert.Equal(t, []string{"b", "a", "c"}, entryIDs(resp.GetResults()))
	assert.Equal(t, "review my code", ext.gotText)
	assert.Equal(t, uint32(3), resp.GetTotalCount())
	assert.Empty(t, resp.GetNextPageToken())
}

func TestSearchAgents_FansOutOneQueryPerSignal(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a")},
		recordCIDs: map[string][]string{"": {"a"}},
	}
	ext := &fakeExtractor{result: resultWith([]string{"code_review", "static_analysis"}, []string{"software_dev"})}

	_, err := searchCtrl(db, ext).SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "q"})
	require.NoError(t, err)

	// One query per signal, each capped — not a single ANDed query.
	assert.Equal(t, 3, db.recordCallCount)

	for _, f := range db.gotRecordQuery {
		assert.Equal(t, int(500), f.Limit, "per-signal fan-out cap")
	}
}

func TestSearchAgents_Pagination(t *testing.T) {
	db := &fakeCatalogDB{
		entries: []*catalogv1.CatalogEntry{entry("a"), entry("b"), entry("c")},
		// One signal returning all three, so rank order is CID-ascending.
		recordCIDs: map[string][]string{"": {"a", "b", "c"}},
	}
	ext := &fakeExtractor{result: resultWith([]string{"code_review"}, nil)}
	ctrl := searchCtrl(db, ext)

	resp1, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "q", PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, entryIDs(resp1.GetResults()))
	assert.Equal(t, uint32(3), resp1.GetTotalCount())
	require.NotEmpty(t, resp1.GetNextPageToken())

	resp2, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{
		Query:     "q",
		PageSize:  2,
		PageToken: resp1.GetNextPageToken(),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"c"}, entryIDs(resp2.GetResults()))
	assert.Empty(t, resp2.GetNextPageToken(), "last page carries no token")
}

func TestSearchAgents_OffsetPastEndIsEmpty(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a")},
		recordCIDs: map[string][]string{"": {"a"}},
	}
	ext := &fakeExtractor{result: resultWith([]string{"code_review"}, nil)}

	resp, err := searchCtrl(db, ext).SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{
		Query:     "q",
		PageToken: encodePageToken(50),
	})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Empty(t, resp.GetNextPageToken())
}

func TestSearchAgents_EmptyExtractionReturnsEmptyPage(t *testing.T) {
	db := &fakeCatalogDB{entries: []*catalogv1.CatalogEntry{entry("a")}}
	ext := &fakeExtractor{result: sdk.Result{}}

	resp, err := searchCtrl(db, ext).SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "banana"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Zero(t, db.recordCallCount, "nothing to search on means no queries")
}

func TestSearchAgents_NoExtractorUnavailable(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "anything"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestSearchAgents_ExtractorErrorUnavailable(t *testing.T) {
	ext := &fakeExtractor{err: errors.New("backend down")}

	_, err := searchCtrl(&fakeCatalogDB{}, ext).SearchAgents(context.Background(),
		&catalogv1.SearchAgentsRequest{Query: "x"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestSearchAgents_InvalidRequests(t *testing.T) {
	ext := &fakeExtractor{result: resultWith([]string{"x"}, nil)}
	ctrl := searchCtrl(&fakeCatalogDB{}, ext)

	tests := []struct {
		name string
		req  *catalogv1.SearchAgentsRequest
	}{
		{"blank query", &catalogv1.SearchAgentsRequest{Query: "  "}},
		{"bad page token", &catalogv1.SearchAgentsRequest{Query: "q", PageToken: "!!!not-base64!!!"}},
		{"query too long", &catalogv1.SearchAgentsRequest{Query: strings.Repeat("x", queryMaxLen+1)}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ctrl.SearchAgents(context.Background(), tc.req)
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestSearchAgents_OnlyCatalogProjectableCandidates(t *testing.T) {
	// GetCatalogEntries only projects records carrying a known catalog module,
	// so the fan-out must apply the same restriction. Otherwise unprojectable
	// records occupy page slots and inflate total_count, then vanish at
	// hydration, leaving short pages.
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a")},
		recordCIDs: map[string][]string{"": {"a"}},
	}
	ext := &fakeExtractor{result: resultWith([]string{"code_review"}, nil)}

	_, err := searchCtrl(db, ext).SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "q"})
	require.NoError(t, err)

	require.NotEmpty(t, db.gotRecordQuery)
	assert.ElementsMatch(t, catalogv1.KnownCatalogModuleNames(), db.gotRecordQuery[0].ModuleNames,
		"fan-out candidates must be restricted to catalog-projectable records")
}

func TestSearchAgents_SearchFailureDegradesRatherThanFails(t *testing.T) {
	// Every signal query fails. The request still succeeds with no results:
	// a backend hiccup should not turn into a 500 for the caller.
	db := &fakeCatalogDB{err: errors.New("db down")}
	ext := &fakeExtractor{result: resultWith([]string{"code_review"}, nil)}

	resp, err := searchCtrl(db, ext).SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "q"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
	assert.Zero(t, resp.GetTotalCount())
}

func TestCatalogEntryCID(t *testing.T) {
	assert.Equal(t, "bafyabc", catalogEntryCID(&catalogv1.CatalogEntry{Identifier: "urn:ai:org.agntcy:cid:bafyabc"}))
	assert.Equal(t, "bare", catalogEntryCID(&catalogv1.CatalogEntry{Identifier: "bare"}))
}
