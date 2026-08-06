// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"testing"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/utils/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeExtractor implements extractor.Extractor with a canned Result.
type fakeExtractor struct {
	result  sdk.Result
	err     error
	gotText string
}

func (f *fakeExtractor) Extract(_ context.Context, text string, _ extractor.ExtractOptions) (extractor.Result, error) {
	f.gotText = text

	if f.err != nil {
		return extractor.Result{}, f.err
	}

	return f.result, nil
}

func (f *fakeExtractor) Close() error { return nil }

// skillResult builds an extractor Result with a single tier-1 skill signal.
func skillResult(name string) sdk.Result {
	return sdk.Result{
		Skills: []sdk.ScoredClass{{Class: sdk.Class{Name: name}, Score: 0.9, Tier: 1}},
	}
}

func entryIDs(entries []*catalogv1.CatalogEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		ids = append(ids, e.GetIdentifier())
	}

	return ids
}

func TestSearchAgents_RelevanceResults(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a"), entry("b"), entry("c")},
		recordCIDs: []string{"c", "a", "b"},
	}
	ext := &fakeExtractor{result: skillResult("code_review")}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	resp, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "review my code"})
	require.NoError(t, err)

	// Results follow the search (CID) order, not the catalog default order.
	assert.Equal(t, []string{"c", "a", "b"}, entryIDs(resp.GetResults()))

	// The extracted skill became a content-search query.
	assert.Equal(t, "review my code", ext.gotText)
	assert.Contains(t, db.gotRecordFilters.SkillNames, "code_review")
	assert.Empty(t, resp.GetNextPageToken())
}

func TestSearchAgents_FilterFacetsCompose(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a")},
		recordCIDs: []string{"a"},
	}
	ext := &fakeExtractor{result: skillResult("code_review")}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	resp, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{
		Query:  "review code",
		Filter: "verified=true",
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)

	// The facet is applied on the hydration query, alongside the CID set.
	require.NotNil(t, db.gotFilters.Verified)
	assert.True(t, *db.gotFilters.Verified)
	assert.Equal(t, []string{"a"}, db.gotFilters.CIDs)
}

func TestSearchAgents_EmptyExtractionFallback(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a"), entry("b")},
		recordCIDs: []string{"unused"},
	}
	ext := &fakeExtractor{result: sdk.Result{}} // no skills or domains
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	resp, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "banana"})
	require.NoError(t, err)
	assert.Len(t, resp.GetResults(), 2)

	// Fallback is a displayName substring match; the content search is not used.
	assert.Equal(t, []string{"*banana*"}, db.gotFilters.Names)
	assert.Empty(t, db.gotRecordFilters.SkillNames, "GetRecordCIDs must not be called on fallback")
}

func TestSearchAgents_NoExtractorUnavailable(t *testing.T) {
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil)

	_, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "anything"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestSearchAgents_ExtractorErrorUnavailable(t *testing.T) {
	ext := &fakeExtractor{err: errors.New("backend down")}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	_, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "x"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestSearchAgents_EmptyQuery(t *testing.T) {
	ext := &fakeExtractor{result: skillResult("x")}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	_, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "  "})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestSearchAgents_Pagination(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a"), entry("b"), entry("c")},
		recordCIDs: []string{"a", "b", "c"},
	}
	ext := &fakeExtractor{result: skillResult("code_review")}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	// Page 1: two of three, so a continuation token is returned.
	resp1, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{Query: "q", PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, entryIDs(resp1.GetResults()))
	require.NotEmpty(t, resp1.GetNextPageToken())

	// Page 2: the remaining one, no further token.
	resp2, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{
		Query:     "q",
		PageSize:  2,
		PageToken: resp1.GetNextPageToken(),
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"c"}, entryIDs(resp2.GetResults()))
	assert.Empty(t, resp2.GetNextPageToken())
}

func TestSearchAgents_UnknownTypeShortCircuits(t *testing.T) {
	db := &fakeCatalogDB{
		entries:    []*catalogv1.CatalogEntry{entry("a")},
		recordCIDs: []string{"a"},
	}
	ext := &fakeExtractor{result: skillResult("x")}
	ctrl := NewAIFinderController("hostId", db, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	resp, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{
		Query:  "q",
		Filter: "type=application/unknown",
	})
	require.NoError(t, err)
	assert.Empty(t, resp.GetResults())
}

func TestSearchAgents_InvalidFilter(t *testing.T) {
	ext := &fakeExtractor{result: skillResult("x")}
	ctrl := NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil, WithExtractor(ext))

	_, err := ctrl.SearchAgents(context.Background(), &catalogv1.SearchAgentsRequest{
		Query:  "q",
		Filter: "type", // no '='
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestCatalogEntryCID(t *testing.T) {
	// Production identifiers are URNs; the CID is the ":cid:" segment.
	assert.Equal(t, "bafyabc", catalogEntryCID(&catalogv1.CatalogEntry{Identifier: "urn:ai:org.agntcy:cid:bafyabc"}))
	// A bare identifier (as used in tests) is returned unchanged.
	assert.Equal(t, "bare", catalogEntryCID(&catalogv1.CatalogEntry{Identifier: "bare"}))
}
