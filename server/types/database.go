// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type DatabaseAPI interface {
	SearchDatabaseAPI
	SyncDatabaseAPI
}

type SearchDatabaseAPI interface {
	// AddRecord adds a new record to the search database.
	AddRecord(record Record) error

	// GetRecords retrieves records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]Record, error)
}

type SyncDatabaseAPI interface {
	// CreateSync creates a new sync object in the database.
	CreateSync(syncObject SyncObject) (string, error)

	// GetSyncByID retrieves a sync object by its ID.
	GetSyncByID(syncID string) (SyncObject, error)

	// GetSyncs retrieves all sync objects.
	GetSyncs() ([]SyncObject, error)

	// UpdateSync updates an existing sync object in the database.
	UpdateSync(syncObject SyncObject) error

	// DeleteSync deletes a sync object by its ID.
	DeleteSync(syncID string) error
}
