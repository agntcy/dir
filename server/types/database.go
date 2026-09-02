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

	// ScanReportDatabaseAPI handles persistence of security scan results.
	ScanReportDatabaseAPI

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

	// CountRecords returns the number of distinct records matching the provided filters.
	CountRecords(opts ...FilterOption) (uint32, error)

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
	// filters, along with whether more results exist beyond the page.
	GetCatalogEntries(opts ...CatalogQueryOption) (entries []*catalogv1.CatalogEntry, hasMore bool, err error)

	// CountCatalogEntries returns the number of distinct records matching the
	// given filters. Limit and offset options are ignored.
	CountCatalogEntries(opts ...CatalogQueryOption) (uint32, error)

	// ListCatalogTags returns distinct catalog tags derived from OASF skills,
	// domains, and record annotations, sorted lexicographically by label.
	ListCatalogTags() ([]*catalogv1.CatalogTag, error)
}

type UsageMetricsDatabaseAPI interface {
	// IncrementPullCount atomically increments the pull counter for a record and
	// updates last_used_at. Creates the row on first use.
	IncrementPullCount(cid string) error

	// IncrementLookupCount atomically increments the lookup counter for a record.
	// Lookups are metadata-only requests (StoreService.Lookup) and carry less
	// weight than pulls in the popularity signal. Creates the row on first use.
	IncrementLookupCount(cid string) error

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

// Scan status values. A row is written for every attempt, so the status is what
// separates "scanned and clean" from "could not be scanned".
const (
	// ScanStatusCompleted means every phase the record declared ran.
	ScanStatusCompleted = "completed"

	// ScanStatusPartial means some phases ran and some did not: the findings
	// are real, the coverage is incomplete.
	ScanStatusPartial = "partial"

	// ScanStatusFailed means no phase produced a result.
	ScanStatusFailed = "failed"
)

// ScannedStatuses are the statuses under which a scanner reached a verdict.
//
// Every safety filter must gate on these: a failed row stores is_safe = false
// to satisfy the NOT NULL column and fail closed, and that false is a
// placeholder, not a verdict.
func ScannedStatuses() []string {
	return []string{ScanStatusCompleted, ScanStatusPartial}
}

// ScanReportObject is a single scanner-run result row, keyed by (record_cid, scanner_type).
type ScanReportObject interface {
	GetRecordCID() string
	GetScannerType() string // "MCP", "SKILL", or "A2A"
	GetIsSafe() bool
	GetMaxSeverity() string // proto Severity name suffix, e.g. "HIGH", "NONE"
	GetUpdatedAt() time.Time

	// GetStatus is one of the ScanStatus* values. Empty is read as completed,
	// so callers predating the field keep their previous meaning.
	GetStatus() string

	// GetFailureReason is a scanner.FailureReason, empty when completed.
	GetFailureReason() string

	// GetFailureDetail is human-readable context for the failure reason.
	GetFailureDetail() string
}

// ScanSchedule controls when a scan result stops suppressing a rescan.
//
// Materialised into a timestamp at write time, because the exponential date
// arithmetic has no portable spelling across the SQLite and PostgreSQL
// backends. Changing these values therefore does not reschedule existing rows;
// each adopts the new policy on its next attempt.
type ScanSchedule struct {
	// FreshFor is how long a result that reached a verdict stays fresh.
	FreshFor time.Duration

	// RetryBase is the delay after a first failure, doubled by each further
	// consecutive failure.
	RetryBase time.Duration

	// RetryMax caps the backoff. Deliberately shorter than FreshFor, so a
	// deployment whose scanner was misconfigured recovers soon after the fix
	// rather than waiting out the full TTL.
	RetryMax time.Duration
}

const (
	DefaultScanFreshFor  = 7 * 24 * time.Hour
	DefaultScanRetryBase = time.Hour
	DefaultScanRetryMax  = 24 * time.Hour
)

// DefaultScanSchedule returns the schedule for callers with no configuration of
// their own, such as ingest recording a report pushed by a peer.
func DefaultScanSchedule() ScanSchedule {
	return ScanSchedule{
		FreshFor:  DefaultScanFreshFor,
		RetryBase: DefaultScanRetryBase,
		RetryMax:  DefaultScanRetryMax,
	}
}

// NextAttempt returns when a row should next be attempted. Zero
// consecutiveFailures means the attempt reached a verdict.
func (s ScanSchedule) NextAttempt(now time.Time, consecutiveFailures int) time.Time {
	if consecutiveFailures <= 0 {
		return now.Add(s.freshFor())
	}

	delay := s.retryBase()
	maxDelay := s.retryMax()

	// Capped inside the loop: a record failing for months would overflow the
	// duration otherwise.
	for range consecutiveFailures - 1 {
		if delay >= maxDelay {
			break
		}

		delay *= 2
	}

	if delay > maxDelay {
		delay = maxDelay
	}

	return now.Add(delay)
}

func (s ScanSchedule) freshFor() time.Duration {
	if s.FreshFor <= 0 {
		return DefaultScanFreshFor
	}

	return s.FreshFor
}

func (s ScanSchedule) retryBase() time.Duration {
	if s.RetryBase <= 0 {
		return DefaultScanRetryBase
	}

	return s.RetryBase
}

func (s ScanSchedule) retryMax() time.Duration {
	if s.RetryMax <= 0 {
		return DefaultScanRetryMax
	}

	return s.RetryMax
}

// ScanReportDatabaseAPI handles persistence and querying of security scan results.
type ScanReportDatabaseAPI interface {
	// UpsertScanReport inserts or updates a scan result row keyed by
	// (record_cid, scanner_type), maintaining the failure counter and retry
	// schedule.
	UpsertScanReport(report ScanReportObject, schedule ScanSchedule) error

	// GetRecordsNeedingScan returns records with no scan result still
	// suppressing a rescan, bounded by ttl.
	GetRecordsNeedingScan(ttl time.Duration) ([]coretypes.Record, error)
}
