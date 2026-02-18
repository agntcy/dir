// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"time"

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

	// NameVerificationDatabaseAPI handles management of name verifications.
	NameVerificationDatabaseAPI

	// Close closes the database connection and releases any resources.
	Close() error

	// IsReady checks if the database connection is ready to serve traffic.
	IsReady(context.Context) bool
}

type SearchDatabaseAPI interface {
	// AddRecord adds a new record to the search database.
	AddRecord(record Record) error

	// GetRecordCIDs retrieves record CIDs based on the provided filters.
	GetRecordCIDs(opts ...FilterOption) ([]string, error)

	// GetRecords retrieves full records based on the provided filters.
	GetRecords(opts ...FilterOption) ([]Record, error)

	// RemoveRecord removes a record from the search database by CID.
	RemoveRecord(cid string) error

	// SetRecordSigned marks a record as signed (called when a public key is attached).
	SetRecordSigned(recordCID string) error
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

type NameVerificationDatabaseAPI interface {
	// CreateNameVerification creates a new name verification for a record.
	CreateNameVerification(verification NameVerificationObject) error

	// UpdateNameVerification updates an existing name verification for a record.
	UpdateNameVerification(verification NameVerificationObject) error

	// GetVerificationByCID retrieves the verification for a record.
	GetVerificationByCID(cid string) (NameVerificationObject, error)

	// GetRecordsNeedingVerification retrieves signed records with verifiable names
	// that either don't have a verification or have an expired verification.
	GetRecordsNeedingVerification(ttl time.Duration) ([]Record, error)
}
