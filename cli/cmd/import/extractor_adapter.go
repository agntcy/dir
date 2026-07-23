// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"context"
	"fmt"

	enricherconfig "github.com/agntcy/dir-importer/enricher/config"
	sdk "github.com/agntcy/oasf-sdk/pkg/extractor"
)

// oasfExtractorAdapter adapts *sdk.Extractor to enricherconfig.RecordExtractor so
// the import pipeline can use the local OASF extractor for in-process enrichment.
type oasfExtractorAdapter struct {
	ext *sdk.Extractor
	// schemaVersion is the OASF version the enriched record is stamped with; the
	// extractor is scoped to it so assigned classes match the schema the server
	// validates the record against.
	schemaVersion string
}

func (a *oasfExtractorAdapter) Extract(ctx context.Context, text string) (enricherconfig.ExtractResult, error) {
	// Scope the extractor to the record's own schema version so it only assigns
	// classes that exist in that version. Latest() would drift ahead of the record's
	// stamped version (e.g. after an OASF release) and the server would reject the
	// newer classes as unknown when validating against the record's schema.
	result, err := a.ext.Extract(ctx, text, sdk.Versions(a.schemaVersion))
	if err != nil {
		return enricherconfig.ExtractResult{}, fmt.Errorf("extract taxonomy: %w", err)
	}

	skills := make([]enricherconfig.TaxonomyClass, len(result.Skills))
	for i, s := range result.Skills {
		skills[i] = enricherconfig.TaxonomyClass{ID: uint32(s.ID), Name: s.Name} //nolint:gosec
	}

	domains := make([]enricherconfig.TaxonomyClass, len(result.Domains))
	for i, d := range result.Domains {
		domains[i] = enricherconfig.TaxonomyClass{ID: uint32(d.ID), Name: d.Name} //nolint:gosec
	}

	return enricherconfig.ExtractResult{Skills: skills, Domains: domains}, nil
}
