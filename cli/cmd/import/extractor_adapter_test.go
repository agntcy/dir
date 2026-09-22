// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"context"
	"testing"

	extractor "github.com/agntcy/dir/utils/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeExtractor records the options it received and returns a canned Result so
// the adapter's option forwarding and class cap can be tested without a model.
type fakeExtractor struct {
	res     extractor.Result
	gotOpts extractor.ExtractOptions
}

func (f *fakeExtractor) Extract(_ context.Context, _ string, opts extractor.ExtractOptions) (extractor.Result, error) {
	f.gotOpts = opts

	return f.res, nil
}

func (f *fakeExtractor) Close() error { return nil }

func scored(id uint64, name string, score float64) sdk.ScoredClass {
	var c sdk.ScoredClass

	c.ID = id
	c.Name = name
	c.Score = score

	return c
}

func TestAdapterForwardsSchemaVersion(t *testing.T) {
	fake := &fakeExtractor{res: sdk.Result{
		Skills:  []sdk.ScoredClass{scored(1, "skill_a", 0.9)},
		Domains: []sdk.ScoredClass{scored(2, "domain_a", 0.8)},
	}}
	a := &oasfExtractorAdapter{ext: fake, schemaVersion: "1.1.0"}

	got, err := a.Extract(context.Background(), "review code")
	require.NoError(t, err)

	assert.Equal(t, []string{"1.1.0"}, fake.gotOpts.Versions)
	// Tiers stays unset so the backend applies DefaultTiers (2).
	assert.Equal(t, 0, fake.gotOpts.Tiers)

	require.Len(t, got.Skills, 1)
	assert.Equal(t, uint32(1), got.Skills[0].ID)
	assert.Equal(t, "skill_a", got.Skills[0].Name)
	require.Len(t, got.Domains, 1)
	assert.Equal(t, "domain_a", got.Domains[0].Name)
}

func TestAdapterCapsSkillsAndDomains(t *testing.T) {
	skills := make([]sdk.ScoredClass, enrichMaxClasses+5)
	for i := range skills {
		skills[i] = scored(uint64(i+1), "skill", 1.0-float64(i)*0.01)
	}

	domains := make([]sdk.ScoredClass, enrichMaxClasses+3)
	for i := range domains {
		domains[i] = scored(uint64(i+100), "domain", 1.0-float64(i)*0.01)
	}

	fake := &fakeExtractor{res: sdk.Result{Skills: skills, Domains: domains}}
	a := &oasfExtractorAdapter{ext: fake, schemaVersion: "1.1.0"}

	got, err := a.Extract(context.Background(), "generic readme")
	require.NoError(t, err)

	require.Len(t, got.Skills, enrichMaxClasses)
	require.Len(t, got.Domains, enrichMaxClasses)
	// Highest-scored classes are kept (the extractor's descending-score prefix).
	assert.Equal(t, uint32(1), got.Skills[0].ID)
	assert.Equal(t, uint32(enrichMaxClasses), got.Skills[enrichMaxClasses-1].ID)
	assert.Equal(t, uint32(100), got.Domains[0].ID)
}
