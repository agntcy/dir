// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "fmt"

type CatalogFilters struct {
	RecordFilters
	MediaTypeFilters []MediaTypeFilter
}

type MediaTypeFilter struct {
	ModuleName        string
	ArtifactMediaType string
}

type CatalogQueryOption interface {
	ApplyCatalog(*CatalogFilters)
}

type CatalogFilterOption func(*CatalogFilters)

func (opt FilterOption) ApplyCatalog(cfg *CatalogFilters) {
	if opt != nil {
		opt(&cfg.RecordFilters)
	}
}

func (opt CatalogFilterOption) ApplyCatalog(cfg *CatalogFilters) {
	if opt != nil {
		opt(cfg)
	}
}

func CatalogFiltersFromOptions(opts ...CatalogQueryOption) (*CatalogFilters, error) {
	cfg := &CatalogFilters{}

	for _, opt := range opts {
		if opt == nil {
			return nil, fmt.Errorf("nil catalog query option provided")
		}

		opt.ApplyCatalog(cfg)
	}

	return cfg, nil
}

func WithMediaTypeFilters(filters ...MediaTypeFilter) CatalogFilterOption {
	return func(cfg *CatalogFilters) {
		cfg.MediaTypeFilters = append(cfg.MediaTypeFilters, filters...)
	}
}
