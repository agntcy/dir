// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package nlsearch

import (
	"context"
	"testing"

	"github.com/agntcy/dir/utils/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExtractor returns a canned Result and records the options it was called
// with, so Decompose's filtering and option forwarding can be tested without a
// real model or server.
type fakeExtractor struct {
	res     extractor.Result
	gotOpts extractor.ExtractOptions
}

func (f *fakeExtractor) Extract(_ context.Context, _ string, opts extractor.ExtractOptions) (extractor.Result, error) {
	f.gotOpts = opts

	return f.res, nil
}

func (f *fakeExtractor) Close() error { return nil }

func skill(name string, tier int, score float64) sdk.ScoredClass {
	return sdk.ScoredClass{Class: sdk.Class{Name: name}, Kind: sdk.KindSkill, Tier: tier, Score: score}
}

func TestDecomposeDefaultKeepsTwoTiersAndWrapsKeywords(t *testing.T) {
	fake := &fakeExtractor{res: sdk.Result{
		Skills: []sdk.ScoredClass{
			skill("skill_t1", 1, 0.9),       // tier 1, above threshold -> kept
			skill("skill_lowscore", 1, 0.1), // below DefaultMinTaxonomyScore -> dropped
			skill("skill_t2", 2, 0.9),       // tier 2 -> kept at the default of two tiers
			skill("skill_t3", 3, 0.9),       // tier 3 -> beyond the default -> dropped
		},
		Domains: []sdk.ScoredClass{
			{Class: sdk.Class{Name: "domain_keep"}, Kind: sdk.KindDomain, Tier: 1, Score: 0.8},
		},
		Keywords: []sdk.Keyword{{Text: "review", Score: 2}},
	}}

	signals, err := Decompose(context.Background(), "review code", fake, extractor.ExtractOptions{Versions: []string{"1.0.0"}})
	require.NoError(t, err)

	// Per-query options are forwarded to the extractor.
	assert.Equal(t, []string{"1.0.0"}, fake.gotOpts.Versions)

	// The two closest tiers survive (skill_t1, skill_t2) plus the domain; the
	// low-score and tier-3 skills are dropped; the keyword is wildcard-wrapped.
	require.Len(t, signals, 4)
	assert.Contains(t, signals, Signal{Type: SignalTypeSkillName, Value: "skill_t1", Score: 0.9})
	assert.Contains(t, signals, Signal{Type: SignalTypeSkillName, Value: "skill_t2", Score: 0.9})
	assert.Contains(t, signals, Signal{Type: SignalTypeDomainName, Value: "domain_keep", Score: 0.8})
	assert.Contains(t, signals, Signal{Type: SignalTypeKeyword, Value: "*review*", Score: 2})
}

func TestDecomposeTiersOverrideNarrows(t *testing.T) {
	fake := &fakeExtractor{res: sdk.Result{
		Skills: []sdk.ScoredClass{
			skill("skill_t1", 1, 0.9),
			skill("skill_t2", 2, 0.8), // beyond an explicit Tiers: 1 -> dropped
		},
	}}

	signals, err := DecomposeWithMinScore(context.Background(), "q", fake, DefaultMinTaxonomyScore,
		extractor.ExtractOptions{Tiers: 1})
	require.NoError(t, err)

	// An explicit tier count is forwarded and narrows below the default of two.
	assert.Equal(t, 1, fake.gotOpts.Tiers)
	require.Len(t, signals, 1)
	assert.Contains(t, signals, Signal{Type: SignalTypeSkillName, Value: "skill_t1", Score: 0.9})
}
