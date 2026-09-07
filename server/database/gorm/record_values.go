// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
	"encoding/json"
	"fmt"
	"sort"

	searchv1 "github.com/agntcy/dir/api/search/v1"
	"github.com/agntcy/dir/server/types"
)

// ListRecordValues returns the distinct values present in the registry for each
// requested field, in the order requested. An empty fields slice returns every
// supported field in canonical order.
//
// Values are registry-wide: no query context and no catalog module restriction
// is applied, so the result is exactly the set of values some record carries.
func (d *DB) ListRecordValues(fields []searchv1.RecordQueryType) ([]types.RecordFieldValues, error) {
	if len(fields) == 0 {
		fields = types.SupportedRecordValueFields()
	}

	result := make([]types.RecordFieldValues, 0, len(fields))

	for _, field := range fields {
		values, err := d.distinctValuesForField(field)
		if err != nil {
			return nil, err
		}

		result = append(result, types.RecordFieldValues{Field: field, Values: values})
	}

	return result, nil
}

// distinctValuesForField dispatches a field to the query that enumerates it.
func (d *DB) distinctValuesForField(field searchv1.RecordQueryType) ([]string, error) {
	switch field { //nolint:exhaustive // unsupported fields are rejected by the default branch
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL_NAME:
		return d.distinctColumn(&Skill{}, "name")
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN_NAME:
		return d.distinctColumn(&Domain{}, "name")
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE_NAME:
		return d.distinctColumn(&Module{}, "name")
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_AUTHOR:
		return d.distinctAuthors()
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION:
		return d.distinctColumn(&Record{}, "schema_version")
	default:
		return nil, fmt.Errorf("unsupported record value field: %s", field)
	}
}

// distinctAuthors returns the distinct authors across all records, sorted
// lexicographically.
//
// Unlike every other supported field, authors are not a column or a child table
// but a JSON array serialized into records.authors, so they cannot be plucked
// with a plain DISTINCT. The array is unnested in Go rather than with SQL JSON
// functions, which keeps one code path across the SQLite and Postgres backends.
// DISTINCT still does most of the work: only distinct JSON payloads are read,
// so records sharing an author list are collapsed before they reach Go.
func (d *DB) distinctAuthors() ([]string, error) {
	var payloads []string

	if err := d.gormDB.
		Model(&Record{}).
		Distinct().
		Where("authors != ?", "").
		Pluck("authors", &payloads).Error; err != nil {
		return nil, fmt.Errorf("list distinct author payloads: %w", err)
	}

	seen := make(map[string]struct{})

	for _, payload := range payloads {
		var authors []string

		// A record written before authors were populated, or one carrying a
		// malformed payload, contributes nothing rather than failing the call.
		if err := json.Unmarshal([]byte(payload), &authors); err != nil {
			logger.Warn("skipping unparsable authors payload", "payload", payload, "error", err)

			continue
		}

		for _, author := range authors {
			if author != "" {
				seen[author] = struct{}{}
			}
		}
	}

	values := make([]string, 0, len(seen))
	for author := range seen {
		values = append(values, author)
	}

	sort.Strings(values)

	return values, nil
}

// distinctColumn plucks the distinct non-empty values of a column, sorted
// lexicographically. Empty values are dropped so that records missing an
// optional field (an unset schema_version, say) do not contribute a blank entry.
func (d *DB) distinctColumn(model any, column string) ([]string, error) {
	var values []string

	if err := d.gormDB.
		Model(model).
		Distinct().
		Where(column+" != ?", "").
		Pluck(column, &values).Error; err != nil {
		return nil, fmt.Errorf("list distinct %s values: %w", column, err)
	}

	sort.Strings(values)

	return values, nil
}
