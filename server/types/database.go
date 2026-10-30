// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
	"time"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	searchv1 "github.com/agntcy/dir/api/search/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
)

//nolint:interfacebloat // one embedded API per persisted concern.
type DatabaseAPI interface {
	// SearchDatabaseAPI handles management of the search database.
	SearchDatabaseAPI

	// SyncDatabaseAPI handles management of the sync database.
	SyncDatabaseAPI

	// PublicationDatabaseAPI handles management of the publication database.
	PublicationDatabaseAPI

	// SignatureVerificationDatabaseAPI handles management of signature verifications.
	SignatureVerificationDatabaseAPI

	// IdentityClaimDatabaseAPI handles persistence of identity and ownership claim results.
	IdentityClaimDatabaseAPI

	// ScanReportDatabaseAPI handles persistence of security scan results.
	ScanReportDatabaseAPI

	// PolicyEvaluationDatabaseAPI handles persistence of content-policy verdicts.
	PolicyEvaluationDatabaseAPI

	// PolicyFactDatabaseAPI reads the stored observations an evaluation policy's
	// meta is projected from.
	PolicyFactDatabaseAPI

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

	// ListRecordSummaries returns one page of lightweight record summaries
	// matching the provided filters, ordered by the given order clauses (most
	// recently indexed first by default), and whether more records follow the page. A limit of
	// 0 or less defaults to DefaultRecordSummaryPageSize; larger than
	// MaxRecordSummaryPageSize is clamped.
	ListRecordSummaries(opts ...FilterOption) ([]*searchv1.RecordSummary, bool, error)

	// ListFilterValues returns the distinct values present in the registry for each
	// requested field, in the order requested. An empty fields slice returns every
	// supported field in canonical order.
	ListFilterValues(fields []searchv1.RecordQueryType) ([]FilterFieldValues, error)

	// GetRecords retrieves full records based on the provided filters.
	GetRecords(opts ...FilterOption) ([]coretypes.Record, error)

	// RemoveRecord removes a record from the search database by CID.
	RemoveRecord(cid string) error

	// SetRecordSigned marks a record as signed (called when a signature is
	// attached). Raising the flag supersedes the verdicts of the policies
	// reading signature facts, in the transaction that stores it.
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

// SignatureVerificationDatabaseAPI persists what verifying a record's
// signatures concluded, one row per signer.
//
// A write here that changes what a policy can read supersedes the verdicts of
// the policies reading signature facts, in the transaction that stores it.
// SetRecordSigned counts as one of those writes.
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

// IdentityClaimDatabaseAPI persists the last verification result of a record's
// identity and ownership claims.
//
// A write here that changes what a policy can read supersedes the verdicts of
// the policies reading identity facts, in the transaction that stores it.
type IdentityClaimDatabaseAPI interface {
	// UpsertIdentityClaim inserts or updates the result keyed by (record CID, role).
	UpsertIdentityClaim(claim IdentityClaimObject) error

	// GetIdentityClaimByCID returns the result for a record's claim of the given
	// role. Returns an error wrapping gorm.ErrIdentityClaimNotFound if none exists.
	GetIdentityClaimByCID(cid, role string) (IdentityClaimObject, error)

	// DeleteIdentityClaim removes the result for a record's claim of the given
	// role, e.g. once the claim itself is gone. Not an error if there is none.
	DeleteIdentityClaim(cid, role string) error
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

// Scan source values. Who produced a scan_reports row.
//
// Empty means the writer did not record one. Rows stored before the column
// existed are empty: they were written both by this node's scanner and by
// ingesting a peer's report, and nothing distinguished the two.
const (
	// ScanSourceLocal means this node's scanner wrote the row.
	ScanSourceLocal = "local"

	// ScanSourceImported means the row was copied from a peer's scan report.
	// It is that peer's assertion, not an observation this node made.
	ScanSourceImported = "imported"
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

	// GetSource is one of the ScanSource* values, or empty when the writer
	// did not record who produced the row.
	GetSource() string

	// GetFindingCount is how many findings this scanner reported. Zero when
	// the writer did not record one, which is what a row stored before the
	// column existed says.
	GetFindingCount() int
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
	// consecutive failure. The reconciler passes its scan interval, so a first
	// failure retries on the next pass and only repeated ones back off.
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

// DefaultScanSchedule returns the all-defaults schedule, for callers with no
// configuration of their own.
func DefaultScanSchedule() ScanSchedule {
	return ScanSchedule{}.withDefaults()
}

// withDefaults fills fields left unset, or set to a nonsensical non-positive
// duration that would otherwise schedule the next attempt in the past.
func (s ScanSchedule) withDefaults() ScanSchedule {
	if s.FreshFor <= 0 {
		s.FreshFor = DefaultScanFreshFor
	}

	if s.RetryBase <= 0 {
		s.RetryBase = DefaultScanRetryBase
	}

	if s.RetryMax <= 0 {
		s.RetryMax = DefaultScanRetryMax
	}

	return s
}

// NextAttempt returns when a row should next be attempted. Zero
// consecutiveFailures means the attempt reached a verdict.
func (s ScanSchedule) NextAttempt(now time.Time, consecutiveFailures int) time.Time {
	s = s.withDefaults()

	if consecutiveFailures <= 0 {
		return now.Add(s.FreshFor)
	}

	// Doubling stops at the cap, because a record failing for months would
	// overflow the duration otherwise.
	delay := s.RetryBase

	for range consecutiveFailures - 1 {
		if delay >= s.RetryMax {
			break
		}

		delay *= 2
	}

	return now.Add(min(delay, s.RetryMax))
}

// ScanReportDatabaseAPI handles persistence and querying of security scan results.
//
// A write here that changes what a policy can read supersedes the verdicts of
// the policies reading scan facts, in the transaction that stores it.
type ScanReportDatabaseAPI interface {
	// UpsertScanReport inserts or updates a scan result row keyed by
	// (record_cid, scanner_type), maintaining the failure counter and retry
	// schedule.
	UpsertScanReport(report ScanReportObject, schedule ScanSchedule) error

	// GetRecordsNeedingScan returns records with no scan result still
	// suppressing a rescan, bounded by ttl.
	GetRecordsNeedingScan(ttl time.Duration) ([]coretypes.Record, error)
}

// PolicyFacts is what this node has stored about one record: the rows an
// evaluation policy's meta is projected from. A record that is not indexed
// comes back as the zero value.
//
// These are the stored rows, not the policy input. The projection drops the
// signature bytes, the public key and the individual scan findings, which a
// policy has no business deciding over.
type PolicyFacts struct {
	// Signed is whether a signature referrer is attached. A signed record with
	// no verification row has not been verified yet, which is a different
	// state from one that was never signed.
	Signed bool

	// IndexedAt is when this node first indexed the record. Zero when the
	// record is not indexed, which is the one case where this node has
	// observed nothing at all about it.
	//
	// This node's own observation, never the created_at a publisher put in the
	// record: when a record was indexed here is not something its author may
	// assert.
	IndexedAt time.Time

	Signatures []SignatureVerificationObject
	Scans      []ScanReportObject
	Claims     []IdentityClaimObject
}

// PolicyFactDatabaseAPI reads the stored observations of one record.
type PolicyFactDatabaseAPI interface {
	// GetPolicyFacts returns the stored observations of cid. A CID that is not
	// indexed returns a zero PolicyFacts and no error.
	GetPolicyFacts(cid string) (PolicyFacts, error)
}

// Policy evaluation status values.
const (
	// PolicyEvalStatusEvaluated means the policy engine reached a verdict:
	// Compliant is authoritative.
	PolicyEvalStatusEvaluated = "evaluated"

	// PolicyEvalStatusFailed means the policy engine errored before reaching
	// a verdict.
	PolicyEvalStatusFailed = "failed"

	// PolicyEvalStatusSuperseded means the verdict was reached, and the
	// metadata it was reached on has since changed. The verdict that is
	// recorded was true when it was written and says nothing about the record
	// now, so it counts as no verdict until the record is evaluated again.
	PolicyEvalStatusSuperseded = "superseded"
)

// Fact groups are the parts of a record's metadata a policy can depend on,
// named after the RecordMeta fields a policy reads, because those names are
// what a policy actually writes: meta.scan, input.meta.signature.
//
// They are the unit invalidation works in. A write knows which group it
// changed, a policy's expressions say which groups it reads, and a verdict is
// superseded only where those meet. The grain is a deliberate compromise: per
// field would spare a policy reading only max_severity from a changed finding
// count, at the cost of a dependency set large enough to be worth storing per
// field, and the groups are what the writers are already divided into.
const (
	// FactGroupScan is meta.scan: what the scanners reported.
	FactGroupScan = "scan"

	// FactGroupSignature is meta.signature: whether the record is signed and
	// what verifying it concluded. The signed flag belongs here, because it is
	// the difference the group reports between an unsigned record and one
	// nothing has verified yet.
	FactGroupSignature = "signature"

	// FactGroupIdentity is meta.identity: what resolving the record's claims
	// of identity and ownership concluded.
	FactGroupIdentity = "identity"

	// FactGroupNone records that a policy version was read and depends on no
	// metadata at all. It is stored rather than left implicit so that "reads
	// nothing" can be told apart from "nothing was ever recorded", which has
	// to be treated as reading everything.
	//
	// Without it the two are the same absence, and a node upgrading would
	// read every policy registered before this existed as depending on
	// nothing, and quietly stop invalidating their verdicts. It is never a
	// group a policy can read: the groups come from RecordMeta's fields, and
	// a name that is not one of those is dropped.
	FactGroupNone = "none"
)

// FactGroups are every group a policy can depend on. It is what a policy whose
// dependencies cannot be read off its source is taken to depend on, so adding
// a group here widens that fallback rather than leaving it stale.
func FactGroups() []string {
	return []string{FactGroupScan, FactGroupSignature, FactGroupIdentity}
}

// EvaluatedPolicyStatuses are the statuses under which a policy evaluator
// reached a verdict.
//
// Every read-path filter must gate on these: a failed row stores compliant =
// false to satisfy the NOT NULL column and fail closed, and that false is a
// placeholder, not a verdict. Mirrors ScannedStatuses for scan reports.
func EvaluatedPolicyStatuses() []string {
	return []string{PolicyEvalStatusEvaluated}
}

// EnforcedPolicy is a policy, at the version currently in force, that a
// record must comply with to be returned by any read. A verdict under a
// different version does not count: until the record is re-evaluated, it is
// not known to comply with this one.
type EnforcedPolicy struct {
	// ID matches PolicyEvaluationObject.GetPolicyID.
	ID string

	// Version matches PolicyEvaluationObject.GetPolicyVersion.
	Version string
}

// PolicyEnforcement is what the server's reads must satisfy: the enforced
// policies, and how each kind of read applies them. See
// policyconfig.EnforcementConfig for which reads each mode covers.
type PolicyEnforcement struct {
	Policies []EnforcedPolicy
	Search   policyconfig.Mode
	Fetch    policyconfig.Mode

	// Pending are the configured policies not enforced yet: no evaluator has
	// registered them (Version is empty), or their verdicts do not yet cover
	// the records and never did.
	Pending []EnforcedPolicy
}

// PolicyGateObserver is told about each record a read by CID excluded, or in
// shadow mode would have excluded.
type PolicyGateObserver interface {
	FetchExcluded(mode policyconfig.Mode)
}

// PolicyEvaluationObject is a single policy verdict row, keyed by
// (record_cid, policy_id).
type PolicyEvaluationObject interface {
	GetRecordCID() string

	// GetPolicyID is a stable identifier of the policy that produced this
	// verdict, e.g. "opa:require-annotation" or "oasf:schema".
	GetPolicyID() string

	// GetPolicyVersion is the content version (or hash) of the policy that
	// produced this verdict. A row whose version no longer matches the
	// policy's current version is stale and is due for re-evaluation, not
	// reinterpreted under the new policy.
	GetPolicyVersion() string

	// GetCompliant is the verdict. Meaningful only when GetStatus is
	// PolicyEvalStatusEvaluated; otherwise it is a fail-closed placeholder
	// and must read as false.
	GetCompliant() bool

	// GetStatus is one of the PolicyEvalStatus* values.
	GetStatus() string

	// GetReason is a human-readable explanation, empty when compliant.
	GetReason() string

	GetUpdatedAt() time.Time
}

// PolicyEvaluationDatabaseAPI handles persistence and querying of
// content-policy verdicts.
type PolicyEvaluationDatabaseAPI interface {
	// UpsertPolicyEvaluation inserts or updates the verdict row keyed by
	// (record_cid, policy_id).
	UpsertPolicyEvaluation(eval PolicyEvaluationObject) error

	// GetPolicyEvaluations retrieves every policy verdict recorded for a
	// record.
	GetPolicyEvaluations(recordCID string) ([]PolicyEvaluationObject, error)

	// GetRecordsNeedingPolicyEvaluation returns records with no evaluated
	// verdict row for policyID at policyVersion — never evaluated against
	// this version, or whose last evaluation failed. Records come in
	// record_cid order after afterCID, at most limit of them; zero returns
	// them all.
	GetRecordsNeedingPolicyEvaluation(policyID, policyVersion, afterCID string, limit int) ([]coretypes.Record, error)

	// RegisterPolicyVersion records that an evaluator is running version of
	// policyID now, so the server can follow the policy's current version,
	// together with the fact groups that version reads.
	//
	// The two are one call because they have to be written together. A
	// version recorded without its groups reads as a policy that depends on
	// no metadata, which is indistinguishable from one that truly does not
	// and would mean its verdicts are never invalidated. Empty factGroups
	// therefore asserts "reads no metadata", and is not the absence of an
	// answer.
	RegisterPolicyVersion(policyID, policyVersion string, factGroups []string) error

	// GetCurrentPolicyVersion returns the version of policyID an evaluator
	// ran most recently, and whether any evaluator has registered it.
	GetCurrentPolicyVersion(policyID string) (string, bool, error)

	// RecordServabilityAPI is the policy gate for reads that fetch by CID.
	RecordServabilityAPI
}

// RecordServabilityAPI is the policy gate for reads that fetch a record by CID
// rather than searching: the store and AI Finder APIs, and the peer RPC.
type RecordServabilityAPI interface {
	// IsRecordServable reports whether a record fetched directly by CID may
	// be returned: always unless fetches enforce the policies, otherwise only
	// if it complies with every enforced policy.
	IsRecordServable(cid string) (bool, error)
}
