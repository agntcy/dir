// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type DatabaseAPI interface {
	// AddRecord adds a new record to the search database.
	AddRecord(record Record) error

	// GetRecords retrieves records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]Record, error)
}
