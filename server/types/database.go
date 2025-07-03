// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type DatabaseAPI interface {
	// AddRecord adds a new agent record to the database.
	AddRecord(record RecordObject) error

	// GetRecords retrieves agent records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]RecordObject, error)

	// CreateSync creates a new sync object in the database.
	CreateSync(remoteURL string) (string, error)

	// GetSyncByID retrieves a sync object by its ID.
	GetSyncByID(syncID string) (SyncObject, error)

	// GetSyncs retrieves all sync objects.
	GetSyncs() ([]SyncObject, error)

	// UpdateSync updates an existing sync object in the database.
	UpdateSync(syncObject SyncObject) error

	// DeleteSync deletes a sync object by its ID.
	DeleteSync(syncID string) error
}
