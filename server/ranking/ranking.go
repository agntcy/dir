// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ranking computes per-record ranking scores for search results.
// Score is a pure function of Signals and Config; callers populate
// Signals from their own data sources (see signals.go).
package ranking

import (
	"math"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/ranking/config"
	"github.com/agntcy/dir/utils/logging"
)

// MaxRankScore is the fixed-point upper bound of composite scores on the wire.
const MaxRankScore = 1000

var logger = logging.Logger("ranking")

// Signals is the bundle of inputs Score consumes. Floats are in [0, 1];
// integer fields are raw counts normalized internally by Score.
type Signals struct {
	QueryRelevance  float64
	Signed          bool
	SigVerified     bool
	NameVerified    bool
	ProviderCount   int
	TaxonomyEntries int
	AgeDays         float64
	SchemaRecency   float64
	// Status surfaces directly as RankExplanation.score_status.
	Status corev1.ScoreStatus
}

// Result is the output of Score, ready to copy onto the wire as
// rank_score and rank_explanation.
type Result struct {
	Score       uint32
	Explanation *corev1.RankExplanation
}

// Scored pairs an arbitrary item with its Result. Item is `any` so the
// ranking package stays independent of storage- and routing-specific
// types; callers cast back to the concrete type they put in.
type Scored struct {
	CID    string
	Item   any
	Result Result
}

// Score is a pure function: each per-signal contribution is normalized
// to [0, 1] and reported via RankExplanation; the composite is the
// weighted sum scaled to [0, MaxRankScore]. Trust is split across the
// three boolean inputs per cfg.TrustSplit.
func Score(signals Signals, cfg config.Config) Result {
	queryRel := clamp01(signals.QueryRelevance)
	trust := trustComponent(signals, cfg)
	popularity := NormalizePopularity(signals.ProviderCount, cfg)
	completeness := NormalizeCompleteness(signals.TaxonomyEntries, cfg)
	schemaRecency := clamp01(signals.SchemaRecency)

	// Without age data, treat freshness as 0; otherwise a missing age
	// would give every defaults-only candidate a 100% freshness windfall.
	freshness := 0.0
	if signals.Status != corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY {
		freshness = FreshnessFromAge(signals.AgeDays, cfg)
	}

	w := cfg.Weights
	composite := w.QueryRelevance*queryRel +
		w.Trust*trust +
		w.Popularity*popularity +
		w.Completeness*completeness +
		w.Freshness*freshness +
		w.SchemaRecency*schemaRecency
	composite = clamp01(composite)

	return Result{
		Score: toFixed(composite),
		Explanation: &corev1.RankExplanation{
			QueryRelevance: toFixed(queryRel),
			Trust:          toFixed(trust),
			Popularity:     toFixed(popularity),
			Completeness:   toFixed(completeness),
			Freshness:      toFixed(freshness),
			SchemaRecency:  toFixed(schemaRecency),
			ProviderCount:  uint32(maxInt(signals.ProviderCount, 0)), //nolint:gosec // bounded by maxInt(>=0)
			ScoreStatus:    signals.Status,
		},
	}
}

// trustComponent combines the three boolean trust inputs into [0, 1]
// using cfg.TrustSplit as weights, so partially trusted records still
// beat unsigned ones.
func trustComponent(s Signals, cfg config.Config) float64 {
	var v float64

	if s.Signed {
		v += cfg.TrustSplit.Signed
	}

	if s.SigVerified {
		v += cfg.TrustSplit.SigVerified
	}

	if s.NameVerified {
		v += cfg.TrustSplit.NameVerified
	}

	return clamp01(v)
}

func toFixed(v float64) uint32 {
	if v <= 0 {
		return 0
	}

	if v >= 1 {
		return MaxRankScore
	}

	return uint32(math.Round(v * MaxRankScore))
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || v <= 0 {
		return 0
	}

	if v >= 1 {
		return 1
	}

	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}
