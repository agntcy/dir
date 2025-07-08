// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"

type DatabaseAPI interface {
	SearchDatabaseAPI
	SyncDatabaseAPI
}

type SearchDatabaseAPI interface {
	// AddRecord adds a new agent record to the database.
	AddRecord(record RecordObject) error

	// GetRecords retrieves agent records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]RecordObject, error)
}

type SyncDatabaseAPI interface {
	// CreateSync creates a new sync object in the database.
	CreateSync(remoteURL string) (string, error)

	// GetSyncByID retrieves a sync object by its ID.
	GetSyncByID(syncID string) (SyncObject, error)

	// GetSyncs retrieves all sync objects.
	GetSyncs() ([]SyncObject, error)

	// GetSyncsByStatus retrieves all sync objects by their status.
	GetSyncsByStatus(status storev1alpha2.SyncStatus) ([]SyncObject, error)

	// UpdateSyncStatus updates an existing sync object in the database.
	UpdateSyncStatus(syncID string, status storev1alpha2.SyncStatus) error

	// DeleteSync deletes a sync object by its ID.
	DeleteSync(syncID string) error
}
