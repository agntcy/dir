// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ranking

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/ranking/config"
	"github.com/agntcy/dir/server/types"
)

// LatestKnownSchemaVersion is the reference point for SchemaRecency.
// Bump this when a new OASF version is supported.
const LatestKnownSchemaVersion = "1.0.0"

const (
	hoursPerDay  = 24
	halfLifeBase = 0.5 // FreshnessFromAge: a record one half-life old scores 0.5.
)

// QueryRelevanceFromMatchScore returns matchScore/numQueries clamped to
// [0, 1]. numQueries == 0 means "no constraints" → 1.0.
func QueryRelevanceFromMatchScore(matchScore, numQueries uint32) float64 {
	if numQueries == 0 {
		return 1.0
	}

	if matchScore >= numQueries {
		return 1.0
	}

	return float64(matchScore) / float64(numQueries)
}

// SigVerified reports whether at least one verification row is "verified".
func SigVerified(verifs []types.SignatureVerificationObject) bool {
	for _, v := range verifs {
		if v != nil && v.GetStatus() == "verified" {
			return true
		}
	}

	return false
}

// NameVerified is true iff the JWKS verification row is "verified".
func NameVerified(verif types.NameVerificationObject) bool {
	return verif != nil && verif.GetStatus() == "verified"
}

// AgeDays returns the record's age in days, trying oasfCreatedAt first
// and falling back to fallbackCreatedAt. Both must be RFC3339. The bool
// reports whether parsing succeeded; on failure the float is 0.
func AgeDays(oasfCreatedAt, fallbackCreatedAt string, now time.Time) (float64, bool) {
	for _, ts := range []string{oasfCreatedAt, fallbackCreatedAt} {
		if ts == "" {
			continue
		}

		t, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			continue
		}

		age := now.Sub(t).Hours() / hoursPerDay
		if age < 0 {
			// Future timestamps are degenerate; treat as 0 days old.
			return 0, true
		}

		return age, true
	}

	return 0, false
}

// SchemaRecency returns version/LatestKnownSchemaVersion clamped to
// [0, 1], where each version is reduced to a single int via semverToInt.
// Unparseable versions return 0.
func SchemaRecency(version string) float64 {
	got, ok := semverToInt(version)
	if !ok {
		return 0
	}

	latest, ok := semverToInt(LatestKnownSchemaVersion)
	if !ok || latest == 0 {
		return 0
	}

	ratio := float64(got) / float64(latest)
	if ratio > 1.0 {
		return 1.0
	}

	if ratio < 0 {
		return 0
	}

	return ratio
}

// TaxonomyEntries returns the raw skills + domains + modules + locators
// count. Score normalizes against cfg.Completeness.SaturationAtEntries.
func TaxonomyEntries(data types.RecordData) int {
	if data == nil {
		return 0
	}

	return len(data.GetSkills()) +
		len(data.GetDomains()) +
		len(data.GetModules()) +
		len(data.GetLocators())
}

// VerificationSignals carries trust signals fetched from the DB. Missing
// rows yield false (the common case for unsigned/pending records).
type VerificationSignals struct {
	SigVerified  bool
	NameVerified bool
}

// FetchVerificationSignals is tolerant: a missing row yields false.
// Only a nil DB is a hard error.
func FetchVerificationSignals(db types.DatabaseAPI, cid string) (VerificationSignals, error) {
	if db == nil {
		return VerificationSignals{}, errors.New("ranking: nil DatabaseAPI")
	}

	var out VerificationSignals

	rows, sigErr := db.GetSignatureVerificationsByRecordCID(cid)
	if sigErr != nil {
		logger.Debug("FetchVerificationSignals: signature lookup failed; defaulting to false", "cid", cid, "error", sigErr)
	} else {
		out.SigVerified = SigVerified(rows)
	}

	if nameVerif, nameErr := db.GetVerificationByCID(cid); nameErr == nil {
		out.NameVerified = NameVerified(nameVerif)
	}

	return out, nil
}

// SignalsFromRecord fills in taxonomy/age/schema from the record body.
// Trust booleans, popularity, and query-relevance are left for the
// caller to set.
func SignalsFromRecord(data types.RecordData, signed bool, now time.Time) Signals {
	if data == nil {
		return Signals{Status: corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY}
	}

	age, ageOK := AgeDays(data.GetCreatedAt(), "", now)

	status := corev1.ScoreStatus_SCORE_STATUS_FULL
	if !ageOK {
		status = corev1.ScoreStatus_SCORE_STATUS_PARTIAL
	}

	return Signals{
		Signed:          signed,
		TaxonomyEntries: TaxonomyEntries(data),
		AgeDays:         age,
		SchemaRecency:   SchemaRecency(data.GetSchemaVersion()),
		Status:          status,
	}
}

// RemoteCandidate is everything the routing service knows about a
// candidate before the local DB is consulted. LocalRecord is nil for
// records only known via remote announcements; in that case per-record
// signals fall back to defaults and Status becomes DEFAULTS_ONLY.
type RemoteCandidate struct {
	CID           string
	MatchScore    uint32
	NumQueries    uint32
	ProviderCount int
	LocalRecord   types.Record
}

// BuildRemoteSignals decides whether a candidate can be fully scored
// (locally indexed and decodable), partially scored (locally indexed
// but undecodable or DB error), or defaults-only (remote-only). db
// may be nil; verification signals fall back to false.
func BuildRemoteSignals(db types.DatabaseAPI, candidate RemoteCandidate, now time.Time) Signals {
	queryRel := QueryRelevanceFromMatchScore(candidate.MatchScore, candidate.NumQueries)

	if candidate.LocalRecord == nil {
		return Signals{
			QueryRelevance: queryRel,
			ProviderCount:  candidate.ProviderCount,
			Status:         corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY,
		}
	}

	data, err := candidate.LocalRecord.GetRecordData()
	if err != nil {
		return Signals{
			QueryRelevance: queryRel,
			ProviderCount:  candidate.ProviderCount,
			Status:         corev1.ScoreStatus_SCORE_STATUS_PARTIAL,
		}
	}

	signed := false
	if sa, ok := candidate.LocalRecord.(types.SignedAware); ok {
		signed = sa.GetSigned()
	} else if sig := data.GetSignature(); sig != nil {
		signed = true
	}

	signals := SignalsFromRecord(data, signed, now)
	signals.QueryRelevance = queryRel
	signals.ProviderCount = candidate.ProviderCount

	if db != nil {
		if verif, vErr := FetchVerificationSignals(db, candidate.CID); vErr == nil {
			signals.SigVerified = verif.SigVerified
			signals.NameVerified = verif.NameVerified
		} else {
			// We have the record but can't enrich trust; downgrade Status.
			signals.Status = corev1.ScoreStatus_SCORE_STATUS_PARTIAL
		}
	}

	return signals
}

// NormalizePopularity saturates a provider count to [0, 1] at
// cfg.Popularity.SaturationAtProviders.
func NormalizePopularity(providerCount int, cfg config.Config) float64 {
	sat := cfg.Popularity.SaturationAtProviders
	if sat <= 0 {
		sat = config.DefaultPopularitySaturation
	}

	if providerCount <= 0 {
		return 0
	}

	if providerCount >= sat {
		return 1.0
	}

	return float64(providerCount) / float64(sat)
}

// NormalizeCompleteness saturates a taxonomy-entry count to [0, 1] at
// cfg.Completeness.SaturationAtEntries.
func NormalizeCompleteness(entries int, cfg config.Config) float64 {
	sat := cfg.Completeness.SaturationAtEntries
	if sat <= 0 {
		sat = config.DefaultCompletenessSaturation
	}

	if entries <= 0 {
		return 0
	}

	if entries >= sat {
		return 1.0
	}

	return float64(entries) / float64(sat)
}

// FreshnessFromAge applies exponential decay: half-life-old → 0.5,
// 2×half-life → 0.25, … Zero or negative age → 1.
func FreshnessFromAge(ageDays float64, cfg config.Config) float64 {
	halfLife := cfg.Freshness.HalfLifeDays
	if halfLife <= 0 {
		halfLife = config.DefaultFreshnessHalfLifeDays
	}

	if ageDays <= 0 {
		return 1.0
	}

	return math.Pow(halfLifeBase, ageDays/float64(halfLife))
}

// semverToInt parses "1.0.0" → 10000, "0.8.1" → 801, etc. Components
// beyond the third are ignored; pre-release/build suffixes are stripped.
// Returns (0, false) for anything unparseable.
func semverToInt(version string) (int, bool) {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if version == "" {
		return 0, false
	}

	parts := strings.SplitN(version, ".", 4) //nolint:mnd

	const maxParts = 3

	if len(parts) > maxParts {
		parts = parts[:maxParts]
	}

	for len(parts) < maxParts {
		parts = append(parts, "0")
	}

	const (
		majorIdx, majorMul = 0, 10000
		minorIdx, minorMul = 1, 100
		patchIdx, patchMul = 2, 1
	)

	var result int

	for i, p := range parts {
		if cut := strings.IndexAny(p, "-+"); cut >= 0 {
			p = p[:cut]
		}

		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, false
		}

		switch i {
		case majorIdx:
			result += n * majorMul
		case minorIdx:
			result += n * minorMul
		case patchIdx:
			result += n * patchMul
		}
	}

	return result, true
}
