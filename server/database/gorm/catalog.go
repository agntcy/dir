// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

func (r *Record) GetAgents(filters ...string) ([]interface{}, error) {
	// Convert filters
	filters = filters

	// Required filters to support:
	// - Appendix A: Filter Expression Syntax
	// - Filter by version, e.g. "version=1.0.0", or "include_versions=true" to include all versions
	// - No version filter, default to latest (last created_at)

	// Fetch
	// run a query here to fetch: records, join with skills, modules, domains, signature
	var records []Record
	err := db.Preload("Skills").Preload("Modules").Preload("Domains").Preload("Annotations").Preload("Signatures").Find(&records).Error
	if err != nil {
		return nil, err
	}

	// Apply conversion to catalog format
	result := make([]interface{}, len(records))
	for i, record := range records {
		catalog, err := record.ToCatalog()
		if err != nil {
			return nil, err
		}
		result[i] = catalog
	}

	return result, nil
}

// returns either a single Entry or a nested Entry
func (r *Record) ToCatalog() (interface{}, error) {
	// Fetch
	return nil, nil
}
