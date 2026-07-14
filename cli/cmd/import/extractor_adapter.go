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
}

func (a *oasfExtractorAdapter) Extract(ctx context.Context, text string) (enricherconfig.ExtractResult, error) {
	// Latest() restricts results to classes present in the newest provisioned OASF version,
	// preventing stale classes from older versions (e.g. 0.8.0) from being assigned to records
	// that will be validated against the current schema.
	result, err := a.ext.Extract(ctx, text, sdk.Latest())
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
