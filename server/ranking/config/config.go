// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config holds tunable parameters for the ranking subsystem.
// Defaults are starting points expected to be tuned with real data.
package config

// Per-signal weights sum to 1.0. Trust is split internally across the
// three boolean inputs (see TrustSplit).
const (
	DefaultQueryRelevanceWeight = 0.30
	DefaultTrustWeight          = 0.30
	DefaultPopularityWeight     = 0.15
	DefaultCompletenessWeight   = 0.05
	DefaultFreshnessWeight      = 0.15
	DefaultSchemaRecencyWeight  = 0.05

	// Trust split: signed / sig-verified / name-verified.
	DefaultTrustSplitSigned       = 0.20
	DefaultTrustSplitSigVerified  = 0.40
	DefaultTrustSplitNameVerified = 0.40

	// DefaultFreshnessHalfLifeDays: a record this old scores 0.5.
	DefaultFreshnessHalfLifeDays = 365

	// Saturation points (signal hits 1.0 at this raw value).
	DefaultPopularitySaturation   = 10
	DefaultCompletenessSaturation = 8

	// DefaultMaxCandidates caps worst-case scoring work per request.
	DefaultMaxCandidates = 10000
)

// Config holds all tunable ranking parameters.
type Config struct {
	Weights      Weights      `json:"weights"      mapstructure:"weights"`
	TrustSplit   TrustSplit   `json:"trust_split"  mapstructure:"trust_split"`
	Freshness    Freshness    `json:"freshness"    mapstructure:"freshness"`
	Popularity   Popularity   `json:"popularity"   mapstructure:"popularity"`
	Completeness Completeness `json:"completeness" mapstructure:"completeness"`
	// MaxCandidates caps the candidate set scored per request.
	MaxCandidates int `json:"max_candidates,omitempty" mapstructure:"max_candidates"`
}

// Weights need not sum to 1.0 (the composite is clamped), but defaults do.
type Weights struct {
	QueryRelevance float64 `json:"query_relevance,omitempty" mapstructure:"query_relevance"`
	Trust          float64 `json:"trust,omitempty"           mapstructure:"trust"`
	Popularity     float64 `json:"popularity,omitempty"      mapstructure:"popularity"`
	Completeness   float64 `json:"completeness,omitempty"    mapstructure:"completeness"`
	Freshness      float64 `json:"freshness,omitempty"       mapstructure:"freshness"`
	SchemaRecency  float64 `json:"schema_recency,omitempty"  mapstructure:"schema_recency"`
}

// TrustSplit subdivides the trust weight; the three fields should sum to 1.0.
type TrustSplit struct {
	Signed       float64 `json:"signed,omitempty"        mapstructure:"signed"`
	SigVerified  float64 `json:"sig_verified,omitempty"  mapstructure:"sig_verified"`
	NameVerified float64 `json:"name_verified,omitempty" mapstructure:"name_verified"`
}

type Freshness struct {
	HalfLifeDays int `json:"half_life_days,omitempty" mapstructure:"half_life_days"`
}

type Popularity struct {
	SaturationAtProviders int `json:"saturation_at_providers,omitempty" mapstructure:"saturation_at_providers"`
}

type Completeness struct {
	SaturationAtEntries int `json:"saturation_at_entries,omitempty" mapstructure:"saturation_at_entries"`
}
