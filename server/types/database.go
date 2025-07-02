// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type DatabaseAPI interface {
	// AddRecord adds a new agent record to the database.
	AddRecord(record RecordObject) error

	// GetRecords retrieves agent records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]RecordObject, error)
}
