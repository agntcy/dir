// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package nlsearch decomposes free-form text into structured search signals
// (OASF skills, domains, and free-text keywords) using an extractor.Extractor.
// It is backend-agnostic: the same decomposition runs whether the extractor is
// the in-process local library or a remote gRPC OASF-SDK server.
package nlsearch

import (
	"context"
	"fmt"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/client/extractor"
)

// SignalType identifies the kind of search signal extracted from free-form text.
type SignalType int

const (
	SignalTypeSkillName  SignalType = iota // maps to RECORD_QUERY_TYPE_SKILL_NAME
	SignalTypeDomainName                   // maps to RECORD_QUERY_TYPE_DOMAIN_NAME
	SignalTypeKeyword                      // fans out to RECORD_QUERY_TYPE_NAME + RECORD_QUERY_TYPE_DESCRIPTION
)

func (t SignalType) String() string {
	switch t {
	case SignalTypeSkillName:
		return "skill"
	case SignalTypeDomainName:
		return "domain"
	case SignalTypeKeyword:
		return "keyword"
	}

	return "unknown"
}

// Signal is a single search signal derived from a natural-language query.
type Signal struct {
	Type  SignalType
	Value string  // query value ready to pass as RecordQuery.Value
	Score float64 // confidence score from the extractor
}

// QueryType returns the RecordQueryType to use when issuing this signal as a
// SearchCIDs request.
func (s Signal) QueryType() searchv1.RecordQueryType {
	switch s.Type {
	case SignalTypeSkillName:
		return searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME
	case SignalTypeDomainName:
		return searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME
	case SignalTypeKeyword:
		return searchv1.RecordQueryType_RECORD_QUERY_TYPE_DESCRIPTION
	}

	return searchv1.RecordQueryType_RECORD_QUERY_TYPE_DESCRIPTION
}

// DefaultMinTaxonomyScore is the minimum semantic similarity score a skill or
// domain must reach to be used as a search signal. Callers can override this
// via DecomposeWithMinScore when a looser threshold is appropriate (e.g. routing
// search, where the DHT has exact taxonomy matches and recall matters more).
const DefaultMinTaxonomyScore = 0.3

// Decompose extracts search signals from free-form text using the given
// extractor, applying DefaultMinTaxonomyScore to filter weak matches. opts
// carries per-query options (e.g. Versions) forwarded to the extractor.
func Decompose(ctx context.Context, text string, ext extractor.Extractor, opts extractor.ExtractOptions) ([]Signal, error) {
	return DecomposeWithMinScore(ctx, text, ext, DefaultMinTaxonomyScore, opts)
}

// DecomposeWithMinScore is like Decompose but uses the given minScore threshold
// instead of DefaultMinTaxonomyScore.
func DecomposeWithMinScore(ctx context.Context, text string, ext extractor.Extractor, minScore float64, opts extractor.ExtractOptions) ([]Signal, error) {
	res, err := ext.Extract(ctx, text, opts)
	if err != nil {
		return nil, fmt.Errorf("extract signals from %q: %w", text, err)
	}

	var signals []Signal

	for _, s := range res.Skills {
		if s.Tier == 1 && s.Score >= minScore {
			signals = append(signals, Signal{
				Type:  SignalTypeSkillName,
				Value: s.Name,
				Score: s.Score,
			})
		}
	}

	for _, d := range res.Domains {
		if d.Tier == 1 && d.Score >= minScore {
			signals = append(signals, Signal{
				Type:  SignalTypeDomainName,
				Value: d.Name,
				Score: d.Score,
			})
		}
	}

	for _, kw := range res.Keywords {
		if kw.Score > 0 {
			signals = append(signals, Signal{
				Type:  SignalTypeKeyword,
				Value: "*" + kw.Text + "*",
				Score: kw.Score,
			})
		}
	}

	return signals, nil
}
