// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
)

// FormatRankedLine returns "[ NNN] value", padding score to four digits
// so a column of rows stays aligned.
func FormatRankedLine(score uint32, value string) string {
	return fmt.Sprintf("[%4d] %s", score, value)
}

// FormatRankExplanation returns a single-line per-signal summary, or
// the empty string when exp is nil.
func FormatRankExplanation(exp *corev1.RankExplanation) string {
	if exp == nil {
		return ""
	}

	const initialCapacity = 128 // 8 fields × ~16 chars

	var b strings.Builder

	b.Grow(initialCapacity)

	fmt.Fprintf(&b, "qr=%d", exp.GetQueryRelevance())
	fmt.Fprintf(&b, " trust=%d", exp.GetTrust())
	fmt.Fprintf(&b, " pop=%d", exp.GetPopularity())
	fmt.Fprintf(&b, " freshness=%d", exp.GetFreshness())
	fmt.Fprintf(&b, " completeness=%d", exp.GetCompleteness())
	fmt.Fprintf(&b, " schema=%d", exp.GetSchemaRecency())
	fmt.Fprintf(&b, " providers=%d", exp.GetProviderCount())
	fmt.Fprintf(&b, " status=%s", shortScoreStatus(exp.GetScoreStatus()))

	return b.String()
}

// shortScoreStatus strips the SCORE_STATUS_ prefix from the enum name.
// Unknown values fall back to s.String() so an old CLI doesn't pretend
// a newer field is missing.
func shortScoreStatus(s corev1.ScoreStatus) string {
	switch s {
	case corev1.ScoreStatus_SCORE_STATUS_UNSPECIFIED:
		return "UNSPECIFIED"
	case corev1.ScoreStatus_SCORE_STATUS_FULL:
		return "FULL"
	case corev1.ScoreStatus_SCORE_STATUS_PARTIAL:
		return "PARTIAL"
	case corev1.ScoreStatus_SCORE_STATUS_DEFAULTS_ONLY:
		return "DEFAULTS_ONLY"
	default:
		return s.String()
	}
}
