// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

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
	gotOpts extractor.ExtractOptions
}

func (f *fakeExtractor) Extract(ctx context.Context, text string, opts extractor.ExtractOptions) (extractor.Result, error) {
	f.gotText = text
	f.gotOpts = opts

	// Both real backends propagate the request context, so the fake does too.
	if err := ctx.Err(); err != nil {
		return extractor.Result{}, fmt.Errorf("fake extract: %w", err)
	}

	if f.err != nil {
		return extractor.Result{}, f.err
	}

	return f.result, nil
}

func (f *fakeExtractor) Close() error { return nil }

func extractCtrl(ext *fakeExtractor) catalogv1.AIFinderServiceServer {
	var opts []AIFinderOption
	if ext != nil {
		opts = append(opts, WithExtractor(ext))
	}

	return NewAIFinderController("hostId", &fakeCatalogDB{}, config.HTTPGatewayConfig{}, nil, opts...)
}

func TestExtractTaxonomy_MapsExtractorResult(t *testing.T) {
	ext := &fakeExtractor{result: sdk.Result{
		Skills: []sdk.ScoredClass{
			{ID: 10304, Name: "natural_language_processing/text_classification", Score: 0.82, Tier: 1},
			{ID: 10201, Name: "natural_language_processing/summarization", Score: 0.44, Tier: 2},
		},
		Domains: []sdk.ScoredClass{
			{ID: 901, Name: "customer_service", Score: 0.71, Tier: 1},
		},
		Keywords: []sdk.Keyword{
			{Text: "support", Score: 0.6},
			{Text: "tickets", Score: 0.5},
		},
	}}

	resp, err := extractCtrl(ext).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: "  analyzing customer support tickets  "})
	require.NoError(t, err)

	assert.Equal(t, "analyzing customer support tickets", ext.gotText, "text is trimmed before extraction")
	assert.Equal(t, extractor.ExtractOptions{}, ext.gotOpts, "unset options let the backend apply DefaultTiers")

	require.Len(t, resp.GetSkills(), 2)
	assert.Equal(t, uint32(10304), resp.GetSkills()[0].GetId())
	assert.Equal(t, "natural_language_processing/text_classification", resp.GetSkills()[0].GetName())
	assert.InDelta(t, 0.82, resp.GetSkills()[0].GetScore(), 1e-9)
	assert.Equal(t, uint32(1), resp.GetSkills()[0].GetTier())
	assert.Equal(t, uint32(2), resp.GetSkills()[1].GetTier(), "score order and tiers are preserved as returned")

	require.Len(t, resp.GetDomains(), 1)
	assert.Equal(t, "customer_service", resp.GetDomains()[0].GetName())

	assert.Equal(t, []string{"support", "tickets"}, resp.GetKeywords())
}

func TestExtractTaxonomy_NoMatchesIsEmptyNotError(t *testing.T) {
	// The endpoint is a passthrough: an extractor that finds nothing is a valid
	// answer, not a failure. Unlike SearchAgents, there is no query to rephrase.
	resp, err := extractCtrl(&fakeExtractor{}).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: "zzzz"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetSkills())
	assert.Empty(t, resp.GetDomains())
	assert.Empty(t, resp.GetKeywords())
}

func TestExtractTaxonomy_ModulesAreNotReturned(t *testing.T) {
	ext := &fakeExtractor{result: sdk.Result{
		Modules: []sdk.ScoredClass{{ID: 5, Name: "runtime/manifest", Score: 0.9, Tier: 1}},
	}}

	resp, err := extractCtrl(ext).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: "runtime manifest"})
	require.NoError(t, err)
	assert.Empty(t, resp.GetSkills())
	assert.Empty(t, resp.GetDomains())
}

func TestExtractTaxonomy_NoExtractorConfigured(t *testing.T) {
	_, err := extractCtrl(nil).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: "anything"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.Contains(t, status.Convert(err).Message(), "extractor")
}

func TestExtractTaxonomy_ExtractorFailure(t *testing.T) {
	ext := &fakeExtractor{err: errors.New("backend down")}

	_, err := extractCtrl(ext).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: "anything"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
	assert.NotContains(t, status.Convert(err).Message(), "backend down", "backend detail stays in the logs")
}

func TestExtractTaxonomy_InvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		req  *catalogv1.ExtractTaxonomyRequest
	}{
		{"nil request", nil},
		{"empty text", &catalogv1.ExtractTaxonomyRequest{}},
		{"whitespace-only text", &catalogv1.ExtractTaxonomyRequest{Text: "   \t\n "}},
		{"text over max length", &catalogv1.ExtractTaxonomyRequest{Text: strings.Repeat("a", textMaxLen+1)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := &fakeExtractor{}

			_, err := extractCtrl(ext).ExtractTaxonomy(context.Background(), tt.req)
			require.Error(t, err)
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Empty(t, ext.gotText, "the extractor is not called for an invalid request")
		})
	}
}

func TestExtractTaxonomy_MaxLengthCountsRunesNotBytes(t *testing.T) {
	// protovalidate's max_len is in Unicode code points, so a byte-counting check
	// would reject non-ASCII text the contract allows — at two bytes per rune
	// here, and three for CJK.
	ext := &fakeExtractor{}
	text := strings.Repeat("é", textMaxLen)
	require.Greater(t, len(text), textMaxLen, "the fixture must be over the limit in bytes")

	_, err := extractCtrl(ext).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: text})
	require.NoError(t, err)
	assert.Equal(t, text, ext.gotText)

	// One rune over is still rejected.
	_, err = extractCtrl(&fakeExtractor{}).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: text + "é"})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestExtractTaxonomy_PreservesContextStatus(t *testing.T) {
	// A caller that hung up or ran out of deadline gets its own status back, not
	// a retryable UNAVAILABLE that blames the backend.
	tests := []struct {
		name string
		ctx  func() (context.Context, context.CancelFunc)
		want codes.Code
	}{
		{"canceled", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			return ctx, func() {}
		}, codes.Canceled},
		{"deadline exceeded", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))

			return ctx, cancel
		}, codes.DeadlineExceeded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.ctx()
			defer cancel()

			_, err := extractCtrl(&fakeExtractor{}).ExtractTaxonomy(ctx,
				&catalogv1.ExtractTaxonomyRequest{Text: "anything"})
			require.Error(t, err)
			assert.Equal(t, tt.want, status.Code(err))
		})
	}
}

func TestExtractTaxonomy_MaxLengthIsCheckedBeforeTrimming(t *testing.T) {
	// Padding must not smuggle an over-length payload past the check.
	ext := &fakeExtractor{}
	text := strings.Repeat(" ", 100) + strings.Repeat("a", textMaxLen) + strings.Repeat(" ", 100)

	_, err := extractCtrl(ext).ExtractTaxonomy(context.Background(),
		&catalogv1.ExtractTaxonomyRequest{Text: text})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
