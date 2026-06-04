// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ranking_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/ranking"
	"github.com/agntcy/dir/server/ranking/config"
)

// defaultCfg returns a Config populated with the package's documented
// defaults. Tests use this instead of duplicating the constants at every
// call site.
func defaultCfg() config.Config {
	return config.Config{
		Weights: config.Weights{
			QueryRelevance: config.DefaultQueryRelevanceWeight,
			Trust:          config.DefaultTrustWeight,
			Popularity:     config.DefaultPopularityWeight,
			Completeness:   config.DefaultCompletenessWeight,
			Freshness:      config.DefaultFreshnessWeight,
			SchemaRecency:  config.DefaultSchemaRecencyWeight,
		},
		TrustSplit: config.TrustSplit{
			Signed:       config.DefaultTrustSplitSigned,
			SigVerified:  config.DefaultTrustSplitSigVerified,
			NameVerified: config.DefaultTrustSplitNameVerified,
		},
		Freshness:     config.Freshness{HalfLifeDays: config.DefaultFreshnessHalfLifeDays},
		Popularity:    config.Popularity{SaturationAtProviders: config.DefaultPopularitySaturation},
		Completeness:  config.Completeness{SaturationAtEntries: config.DefaultCompletenessSaturation},
		MaxCandidates: config.DefaultMaxCandidates,
	}
}

// TestDefaultConfig sanity-checks that the default weights sum to ~1.0
// and that the trust split is internally consistent. If anyone tweaks
// the defaults, this test reminds them to keep the invariant.
func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	weightSum := cfg.Weights.QueryRelevance +
		cfg.Weights.Trust +
		cfg.Weights.Popularity +
		cfg.Weights.Completeness +
		cfg.Weights.Freshness +
		cfg.Weights.SchemaRecency

	if delta := absFloat(weightSum - 1.0); delta > 0.001 {
		t.Errorf("default weights sum to %f, want 1.0 (delta %f)", weightSum, delta)
	}

	trustSum := cfg.TrustSplit.Signed +
		cfg.TrustSplit.SigVerified +
		cfg.TrustSplit.NameVerified

	if delta := absFloat(trustSum - 1.0); delta > 0.001 {
		t.Errorf("default trust split sums to %f, want 1.0 (delta %f)", trustSum, delta)
	}
}

// goldenCase is one row of the Score fixtures. Want is checked exactly
// (no float comparison), so any change to the scoring math or default
// weights forces an explicit, reviewed update.
type goldenCase struct {
	name    string
	signals ranking.Signals
	want    uint32
	wantExp *corev1.RankExplanation
}

// TestScoreGoldenFixtures locks the scoring math against hand-computed
// expectations. Each fixture's composite score and per-signal sub-scores
// must be reproduced exactly. When weights or normalization change, the
// expected values must be recomputed deliberately.
//
// All cases use defaultCfg():
//
//	Weights: QR=0.30, Trust=0.30, Pop=0.15, Compl=0.05, Fresh=0.15, Schema=0.05
//	TrustSplit: signed=0.20, sig_verified=0.40, name_verified=0.40
//	Freshness half-life=365d, Popularity saturation=10, Completeness saturation=8
func TestScoreGoldenFixtures(t *testing.T) {
	t.Parallel()

	cases := []goldenCase{
		{
			name: "perfect record",
			signals: ranking.Signals{
				QueryRelevance:  1.0,
				Signed:          true,
				SigVerified:     true,
				NameVerified:    true,
				ProviderCount:   15, // saturates at 10
				TaxonomyEntries: 12, // saturates at 8
				AgeDays:         0,
				SchemaRecency:   1.0,
				Status:          corev1.ScoreStatus_SCORE_STATUS_FULL,
			},
			want: 1000,
			wantExp: &corev1.RankExplanation{
				QueryRelevance: 1000,
				Trust:          1000,
				Popularity:     1000,
				Completeness:   1000,
				Freshness:      1000,
				SchemaRecency:  1000,
				ProviderCount:  15,
				ScoreStatus:    corev1.ScoreStatus_SCORE_STATUS_FULL,
			},
		},
		{
			name:    "all zeros",
			signals: ranking.Signals{Status: corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY},
			want:    0,
			wantExp: &corev1.RankExplanation{
				ScoreStatus: corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY,
			},
		},
		{
			// Composite =
			//   0.30 * 1.0     // query
			// + 0.30 * 0.60    // trust (signed+sig_verified, not name)
			// + 0.15 * 0.50    // popularity (5/10)
			// + 0.05 * 0.50    // completeness (4/8)
			// + 0.15 * 0.50    // freshness (e^-(365/365 * ln2) = 0.5)
			// + 0.05 * 0.08    // schema_recency
			// = 0.300 + 0.180 + 0.075 + 0.025 + 0.075 + 0.004 = 0.659
			name: "trusted but stale",
			signals: ranking.Signals{
				QueryRelevance:  1.0,
				Signed:          true,
				SigVerified:     true,
				NameVerified:    false,
				ProviderCount:   5,
				TaxonomyEntries: 4,
				AgeDays:         365,
				SchemaRecency:   0.08,
				Status:          corev1.ScoreStatus_SCORE_STATUS_FULL,
			},
			want: 659,
			wantExp: &corev1.RankExplanation{
				QueryRelevance: 1000,
				Trust:          600,
				Popularity:     500,
				Completeness:   500,
				Freshness:      500,
				SchemaRecency:  80,
				ProviderCount:  5,
				ScoreStatus:    corev1.ScoreStatus_SCORE_STATUS_FULL,
			},
		},
		{
			// Composite =
			//   0.30 * 0.5   // query (1 of 2 matched)
			// + 0.30 * 0     // trust
			// + 0.15 * 0     // popularity
			// + 0.05 * 1.0   // completeness
			// + 0.15 * 1.0   // freshness
			// + 0.05 * 1.0   // schema
			// = 0.15 + 0.05 + 0.15 + 0.05 = 0.40
			name: "fresh but untrusted",
			signals: ranking.Signals{
				QueryRelevance:  0.5,
				TaxonomyEntries: 8,
				AgeDays:         0,
				SchemaRecency:   1.0,
				Status:          corev1.ScoreStatus_SCORE_STATUS_FULL,
			},
			want: 400,
			wantExp: &corev1.RankExplanation{
				QueryRelevance: 500,
				Trust:          0,
				Popularity:     0,
				Completeness:   1000,
				Freshness:      1000,
				SchemaRecency:  1000,
				ProviderCount:  0,
				ScoreStatus:    corev1.ScoreStatus_SCORE_STATUS_FULL,
			},
		},
		{
			// Out-of-range inputs (negative, >1) clamp safely. Status is
			// PARTIAL, so freshness still fires on the zero-valued
			// AgeDays (treated as "brand new today").
			//
			// Composite = 0.30 * 1 (QR clamped) + 0.15 * 1 (freshness) = 0.45.
			name: "out-of-range inputs clamp",
			signals: ranking.Signals{
				QueryRelevance: 5.0,  // clamps to 1
				SchemaRecency:  -1.0, // clamps to 0
				ProviderCount:  -3,   // clamps to 0
				Status:         corev1.ScoreStatus_SCORE_STATUS_PARTIAL,
			},
			want: 450,
			wantExp: &corev1.RankExplanation{
				QueryRelevance: 1000,
				Trust:          0,
				Popularity:     0,
				Completeness:   0,
				Freshness:      1000,
				SchemaRecency:  0,
				ProviderCount:  0,
				ScoreStatus:    corev1.ScoreStatus_SCORE_STATUS_PARTIAL,
			},
		},
	}

	cfg := defaultCfg()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := ranking.Score(tc.signals, cfg)

			if got.Score != tc.want {
				t.Errorf("Score = %d, want %d", got.Score, tc.want)
			}

			assertExplanationEqual(t, got.Explanation, tc.wantExp)
		})
	}
}

// TestScoreFreshnessHalfLife pins the half-life math: a record exactly
// half-life days old must get freshness sub-score 500 (0.5 * 1000).
func TestScoreFreshnessHalfLife(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	s := ranking.Signals{AgeDays: float64(cfg.Freshness.HalfLifeDays)}

	got := ranking.Score(s, cfg)
	if got.Explanation.GetFreshness() != 500 {
		t.Errorf("freshness at half-life = %d, want 500", got.Explanation.GetFreshness())
	}
}

// TestScoreStatusPropagates ensures the Status that the caller put on
// Signals appears verbatim in the returned RankExplanation.
func TestScoreStatusPropagates(t *testing.T) {
	t.Parallel()

	for _, status := range []corev1.ScoreStatus{
		corev1.ScoreStatus_SCORE_STATUS_FULL,
		corev1.ScoreStatus_SCORE_STATUS_PARTIAL,
		corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY,
	} {
		got := ranking.Score(ranking.Signals{Status: status}, defaultCfg())
		if got.Explanation.GetScoreStatus() != status {
			t.Errorf("status %v did not propagate; got %v", status, got.Explanation.GetScoreStatus())
		}
	}
}

func assertExplanationEqual(t *testing.T, got, want *corev1.RankExplanation) {
	t.Helper()

	if got == nil {
		t.Fatal("explanation is nil")
	}

	if got.GetQueryRelevance() != want.GetQueryRelevance() {
		t.Errorf("query_relevance = %d, want %d", got.GetQueryRelevance(), want.GetQueryRelevance())
	}

	if got.GetTrust() != want.GetTrust() {
		t.Errorf("trust = %d, want %d", got.GetTrust(), want.GetTrust())
	}

	if got.GetPopularity() != want.GetPopularity() {
		t.Errorf("popularity = %d, want %d", got.GetPopularity(), want.GetPopularity())
	}

	if got.GetCompleteness() != want.GetCompleteness() {
		t.Errorf("completeness = %d, want %d", got.GetCompleteness(), want.GetCompleteness())
	}

	if got.GetFreshness() != want.GetFreshness() {
		t.Errorf("freshness = %d, want %d", got.GetFreshness(), want.GetFreshness())
	}

	if got.GetSchemaRecency() != want.GetSchemaRecency() {
		t.Errorf("schema_recency = %d, want %d", got.GetSchemaRecency(), want.GetSchemaRecency())
	}

	if got.GetProviderCount() != want.GetProviderCount() {
		t.Errorf("provider_count = %d, want %d", got.GetProviderCount(), want.GetProviderCount())
	}

	if got.GetScoreStatus() != want.GetScoreStatus() {
		t.Errorf("score_status = %v, want %v", got.GetScoreStatus(), want.GetScoreStatus())
	}
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}

	return f
}
