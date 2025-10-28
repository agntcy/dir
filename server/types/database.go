// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
)

type DatabaseAPI interface {
	// SearchDatabaseAPI handles management of the search database.
	SearchDatabaseAPI

	// SyncDatabaseAPI handles management of the sync database.
	SyncDatabaseAPI

	// PublicationDatabaseAPI handles management of the publication database.
	PublicationDatabaseAPI

	// IsReady checks if the database connection is ready to serve traffic.
	IsReady(context.Context) bool
}

type SearchDatabaseAPI interface {
	// AddRecord adds a new record to the search database.
	AddRecord(record Record) error

	// GetRecords retrieves records based on the provided RecordFilters.
	GetRecords(opts ...FilterOption) ([]Record, error)

	// GetRecordCIDs retrieves only record CIDs based on the provided filters.
	// This is more efficient than GetRecords when only CIDs are needed.
	GetRecordCIDs(opts ...FilterOption) ([]string, error)

	// RemoveRecord removes a record from the search database by CID.
	RemoveRecord(cid string) error
}

type SyncDatabaseAPI interface {
	// CreateSync creates a new sync object in the database.
	CreateSync(remoteURL string, cids []string) (string, error)

	// GetSyncByID retrieves a sync object by its ID.
	GetSyncByID(syncID string) (SyncObject, error)

	// GetSyncs retrieves all sync objects.
	GetSyncs(offset, limit int) ([]SyncObject, error)

	// GetSyncsByStatus retrieves all sync objects by their status.
	GetSyncsByStatus(status storev1.SyncStatus) ([]SyncObject, error)

	// UpdateSyncStatus updates an existing sync object in the database.
	UpdateSyncStatus(syncID string, status storev1.SyncStatus) error

	// UpdateSyncRemoteRegistry updates the remote registry of a sync object.
	UpdateSyncRemoteRegistry(syncID string, remoteRegistry string) error

	// GetSyncRemoteRegistry retrieves the remote registry of a sync object.
	GetSyncRemoteRegistry(syncID string) (string, error)

	// DeleteSync deletes a sync object by its ID.
	DeleteSync(syncID string) error
}

type PublicationDatabaseAPI interface {
	// CreatePublication creates a new publication object in the database.
	CreatePublication(request *routingv1.PublishRequest) (string, error)

	// GetPublicationByID retrieves a publication object by its ID.
	GetPublicationByID(publicationID string) (PublicationObject, error)

	// GetPublications retrieves all publication objects.
	GetPublications(offset, limit int) ([]PublicationObject, error)

	// GetPublicationsByStatus retrieves all publication objects by their status.
	GetPublicationsByStatus(status routingv1.PublicationStatus) ([]PublicationObject, error)

	// UpdatePublicationStatus updates an existing publication object's status in the database.
	UpdatePublicationStatus(publicationID string, status routingv1.PublicationStatus) error

	// DeletePublication deletes a publication object by its ID.
	DeletePublication(publicationID string) error
}
