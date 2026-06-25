// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
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

	// SignatureVerificationDatabaseAPI handles management of signature verifications.
	SignatureVerificationDatabaseAPI

	// CatalogDatabaseAPI handles deterministic browsing of AI Catalog entries.
	CatalogDatabaseAPI

	// UsageMetricsDatabaseAPI handles per-record usage counters for popularity ranking.
	UsageMetricsDatabaseAPI

	// Close closes the database connection and releases any resources.
	Close() error

	// IsReady checks if the database connection is ready to serve traffic.
	IsReady(context.Context) bool
}

type SearchDatabaseAPI interface {
	// AddRecord adds a new record to the search database.
	AddRecord(record coretypes.Record) error

	// GetRecordCIDs retrieves record CIDs based on the provided filters.
	GetRecordCIDs(opts ...FilterOption) ([]string, error)

	// GetRecords retrieves full records based on the provided filters.
	GetRecords(opts ...FilterOption) ([]coretypes.Record, error)

	// RemoveRecord removes a record from the search database by CID.
	RemoveRecord(cid string) error

	// SetRecordSigned marks a record as signed (called when a signature is attached).
	SetRecordSigned(recordCID string) error
}

type SyncDatabaseAPI interface {
	// CreateSync creates a new sync object in the database.
	CreateSync(remoteURL string, cids []string, remoteRegistryURL string, repositoryName string) (string, error)

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
	GetRecordsNeedingVerification(ttl time.Duration) ([]coretypes.Record, error)
}

// CatalogDatabaseAPI exposes the deterministic-browsing query backing the
// AI Finder GET /v1/agents endpoint.
type CatalogDatabaseAPI interface {
	// GetCatalogEntries returns the AI Catalog entries matching the given
	// record filters, along with whether more results exist beyond the page.
	GetCatalogEntries(opts ...FilterOption) (entries []*catalogv1.CatalogEntry, hasMore bool, err error)
}

type UsageMetricsDatabaseAPI interface {
	// IncrementPullCount atomically increments the pull counter for a record and
	// updates last_used_at. Creates the row on first use.
	IncrementPullCount(cid string) error

	// SetProviderCount sets the provider count gauge for a record (point-in-time,
	// refreshed by the reconciler). Creates the row if it does not exist.
	SetProviderCount(cid string, count uint32) error

	// GetUsageMetrics returns the usage metrics for a record. Returns a zero-value
	// result (not an error) if no usage has been recorded yet.
	GetUsageMetrics(cid string) (UsageMetricsObject, error)
}

type SignatureVerificationDatabaseAPI interface {
	// CreateSignatureVerification creates a new signature verification row (one per signer).
	CreateSignatureVerification(verification SignatureVerificationObject) error

	// UpdateSignatureVerification updates an existing row by (record_cid, signer fields).
	UpdateSignatureVerification(verification SignatureVerificationObject) error

	// UpsertSignatureVerification inserts or updates a row keyed by (record_cid, signer fields).
	UpsertSignatureVerification(verification SignatureVerificationObject) error

	// GetSignatureVerificationsByRecordCID returns all signature verification rows for a record.
	GetSignatureVerificationsByRecordCID(recordCID string) ([]SignatureVerificationObject, error)

	// GetRecordsNeedingSignatureVerification returns signed records that have no verification or expired verification.
	GetRecordsNeedingSignatureVerification(ttl time.Duration) ([]coretypes.Record, error)

	// InvalidateSignatureVerificationsForRecord removes all cached verification rows for a record so the reconciler will re-verify it (e.g. when a new signature or public key referrer is pushed).
	InvalidateSignatureVerificationsForRecord(recordCID string) error
}
