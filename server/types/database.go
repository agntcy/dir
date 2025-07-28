// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1alpha2 "github.com/agntcy/dir/api/store/v1alpha2"
)

type DatabaseAPI interface {
	SearchDatabaseAPI
	SyncDatabaseAPI
}

type SearchDatabaseAPI interface {
	// AddRecord adds a new record to the search database.
	AddRecord(record Record) error

	// GetRecords retrieves records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]Record, error)

	// GetRecordRefs retrieves only record references (CIDs) based on the provided filters.
	// This is more efficient than GetRecords when only CIDs are needed.
	GetRecordRefs(opts ...FilterOption) ([]*corev1.RecordRef, error)
}

type SyncDatabaseAPI interface {
	// CreateSync creates a new sync object in the database.
	CreateSync(remoteURL string) (string, error)

	// GetSyncByID retrieves a sync object by its ID.
	GetSyncByID(syncID string) (SyncObject, error)

	// GetSyncs retrieves all sync objects.
	GetSyncs(offset, limit int) ([]SyncObject, error)

	// GetSyncsByStatus retrieves all sync objects by their status.
	GetSyncsByStatus(status storev1alpha2.SyncStatus) ([]SyncObject, error)

	// UpdateSyncStatus updates an existing sync object in the database.
	UpdateSyncStatus(syncID string, status storev1alpha2.SyncStatus) error

	// UpdateSyncRemoteRegistry updates the remote registry of a sync object.
	UpdateSyncRemoteRegistry(syncID string, remoteRegistry string) error

	// GetSyncRemoteRegistry retrieves the remote registry of a sync object.
	GetSyncRemoteRegistry(syncID string) (string, error)

	// DeleteSync deletes a sync object by its ID.
	DeleteSync(syncID string) error
}
