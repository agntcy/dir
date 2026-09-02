// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package gorm

import (
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
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_VERSION:
		return d.distinctColumn(&Record{}, "version")
	case searchv1.RecordQueryType_RECORD_QUERY_TYPE_SCHEMA_VERSION:
		return d.distinctColumn(&Record{}, "schema_version")
	default:
		return nil, fmt.Errorf("unsupported record value field: %s", field)
	}
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
