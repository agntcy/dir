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

	// SignatureVerificationDatabaseAPI handles caching of signature verification results.
	SignatureVerificationDatabaseAPI

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
	CreateSync(remoteURL string, remoteRegistryAddress string, cids []string, requiresRegsync bool) (string, error)

	// GetSyncByID retrieves a sync object by its ID.
	GetSyncByID(syncID string) (SyncObject, error)

	// GetSyncs retrieves all sync objects.
	GetSyncs(offset, limit int) ([]SyncObject, error)

	// GetSyncsByStatus retrieves all sync objects by their status.
	GetSyncsByStatus(status storev1.SyncStatus) ([]SyncObject, error)

	// GetZotSyncsByStatus retrieves sync objects that use Zot-to-Zot sync (requires_regsync = false).
	GetZotSyncsByStatus(status storev1.SyncStatus) ([]SyncObject, error)

	// GetRegsyncSyncsByStatus retrieves sync objects that require regsync (requires_regsync = true).
	GetRegsyncSyncsByStatus(status storev1.SyncStatus) ([]SyncObject, error)

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

// SignatureVerificationDatabaseAPI handles caching of signature verification results.
// Each signature on a record is cached individually, keyed by record CID + signature digest.
type SignatureVerificationDatabaseAPI interface {
	// CreateSignatureVerification creates a new signature verification cache entry.
	CreateSignatureVerification(input SignatureVerificationInput) error

	// GetSignatureVerification retrieves a cached verification by record CID and signature digest.
	GetSignatureVerification(recordCID, signatureDigest string) (SignatureVerificationObject, error)

	// GetSignatureVerificationsByRecord retrieves all cached verifications for a record.
	GetSignatureVerificationsByRecord(recordCID string) ([]SignatureVerificationObject, error)

	// DeleteSignatureVerification deletes a cached verification by record CID and signature digest.
	DeleteSignatureVerification(recordCID, signatureDigest string) error

	// DeleteSignatureVerificationsByRecord deletes all cached verifications for a record.
	DeleteSignatureVerificationsByRecord(recordCID string) error
}
