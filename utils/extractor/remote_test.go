// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"testing"

	extractorv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/extractor/v1"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

// fakeExtractorClient is a stand-in ExtractorServiceClient that records the
// request it received and returns a canned response, so we can assert the
// remoteExtractor's request construction and response mapping in isolation.
type fakeExtractorClient struct {
	resp   *extractorv1.ExtractResponse
	gotReq *extractorv1.ExtractRequest
}

func (f *fakeExtractorClient) Extract(_ context.Context, in *extractorv1.ExtractRequest, _ ...grpc.CallOption) (*extractorv1.ExtractResponse, error) {
	f.gotReq = in

	return f.resp, nil
}

func TestRemoteExtractorMapsResponseToResult(t *testing.T) {
	resp := extractorv1.ExtractResponse_builder{
		Skills: []*extractorv1.ScoredClass{
			extractorv1.ScoredClass_builder{
				Id:       10304,
				Name:     "natural_language_processing/quality_assurance",
				Caption:  "Quality Assurance",
				Kind:     extractorv1.ClassType_CLASS_TYPE_SKILL,
				Versions: []string{"1.0.0"},
				Score:    0.91,
				Tier:     1,
			}.Build(),
		},
		Domains: []*extractorv1.ScoredClass{
			extractorv1.ScoredClass_builder{
				Name:  "technology/software_engineering",
				Kind:  extractorv1.ClassType_CLASS_TYPE_DOMAIN,
				Score: 0.7,
				Tier:  1,
			}.Build(),
		},
		Keywords: []*extractorv1.Keyword{
			extractorv1.Keyword_builder{Text: "review", Score: 0.5}.Build(),
		},
	}.Build()

	fake := &fakeExtractorClient{resp: resp}
	r := &remoteExtractor{client: fake}

	got, err := r.Extract(context.Background(), "a skill for reviewing code", ExtractOptions{Versions: []string{"1.0.0"}})
	require.NoError(t, err)

	// The free text and version pins are forwarded to the server.
	assert.Equal(t, "a skill for reviewing code", fake.gotReq.GetText())
	assert.Equal(t, []string{"1.0.0"}, fake.gotReq.GetVersions())

	// An unset Tiers is sent as DefaultTiers (not 0, which the server reads as 1).
	assert.Equal(t, uint32(DefaultTiers), fake.gotReq.GetTiers())

	// Skills map with kind, score, tier, and identity fields preserved.
	require.Len(t, got.Skills, 1)
	assert.Equal(t, "natural_language_processing/quality_assurance", got.Skills[0].Name)
	assert.Equal(t, uint64(10304), got.Skills[0].ID)
	assert.Equal(t, sdk.KindSkill, got.Skills[0].Kind)
	assert.InDelta(t, 0.91, got.Skills[0].Score, 1e-9)
	assert.Equal(t, 1, got.Skills[0].Tier)
	assert.Equal(t, []string{"1.0.0"}, got.Skills[0].Versions)

	// Domains carry the domain kind.
	require.Len(t, got.Domains, 1)
	assert.Equal(t, sdk.KindDomain, got.Domains[0].Kind)

	// Keywords map text and score.
	require.Len(t, got.Keywords, 1)
	assert.Equal(t, "review", got.Keywords[0].Text)
	assert.InDelta(t, 0.5, got.Keywords[0].Score, 1e-9)
}
