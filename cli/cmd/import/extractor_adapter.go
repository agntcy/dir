// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"context"
	"fmt"
	"time"

	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	extractor "github.com/agntcy/dir/utils/extractor"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// extractTimeout bounds a single Extract call so a hung or unreachable remote
// extractor server can't stall the import indefinitely.
const extractTimeout = 30 * time.Second

// enrichMaxClasses is the maximum number of skills and of domains written onto
// an imported record. Typical imports already land in the 2–6 range; this is
// the safety bound for a flat high-score plateau.
const enrichMaxClasses = 10

// oasfExtractorAdapter adapts a utils/extractor.Extractor to
// enricherconfig.RecordExtractor so the import pipeline can enrich records with
// either the local in-process extractor or a remote OASF-SDK server.
type oasfExtractorAdapter struct {
	ext extractor.Extractor
	// schemaVersion is the OASF version the enriched record is stamped with; the
	// extractor is scoped to it so assigned classes match the schema the server
	// validates the record against.
	schemaVersion string
}

func (a *oasfExtractorAdapter) Extract(ctx context.Context, text string) (enricherconfig.ExtractResult, error) {
	ctx, cancel := context.WithTimeout(ctx, extractTimeout)
	defer cancel()

	// Scope the extractor to the record's own schema version so it only assigns
	// classes that exist in that version. Latest() would drift ahead of the record's
	// stamped version (e.g. after an OASF release) and the server would reject the
	// newer classes as unknown when validating against the record's schema.
	result, err := a.ext.Extract(ctx, text, extractor.ExtractOptions{Versions: []string{a.schemaVersion}})
	if err != nil {
		return enricherconfig.ExtractResult{}, fmt.Errorf("extract taxonomy: %w", err)
	}

	return enricherconfig.ExtractResult{
		Skills:  toTaxonomyClasses(result.Skills),
		Domains: toTaxonomyClasses(result.Domains),
	}, nil
}

func toTaxonomyClasses(in []sdk.ScoredClass) []enricherconfig.TaxonomyClass {
	if len(in) > enrichMaxClasses {
		in = in[:enrichMaxClasses]
	}

	out := make([]enricherconfig.TaxonomyClass, len(in))
	for i, c := range in {
		out[i] = enricherconfig.TaxonomyClass{ID: uint32(c.ID), Name: c.Name} //nolint:gosec
	}

	return out
}
