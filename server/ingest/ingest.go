// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package ingest provides a single, authoritative code path for persisting
// records and referrers with full parity to a normal push:
// content store + search index + referrer-derived database state.
//
// It is used by the gRPC store controller and by DHT-based autosync so that
// content received from any source is stored and indexed identically.
package ingest

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/identity"
	"github.com/agntcy/dir/server/identity/spiffe"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logger = logging.Logger("ingest")

// Ingestor persists records and referrers with full parity to a normal push.
//
// Implementations store the content in the object store and keep the search
// database (and referrer-derived state such as the signed flag and scan report
// summaries) consistent, mirroring the behavior of the gRPC store controller.
type Ingestor interface {
	// ImportRecord pushes a record to the content store and adds it to the
	// search index. Search indexing failures are logged but not fatal (the
	// content store is the source of truth), matching the store controller.
	ImportRecord(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error)

	// ImportReferrer pushes a referrer to the content store and applies the
	// referrer-derived database side-effects for its type:
	//   - Signature  -> mark the record signed + invalidate cached verifications
	//   - PublicKey   -> invalidate cached verifications
	//   - ScanReport  -> upsert the scan report summary row
	// Database side-effects are logged but not fatal (the referrer is already
	// stored), matching the store controller.
	ImportReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) (*corev1.ReferrerRef, error)
}

type ingestor struct {
	store            types.StoreAPI
	db               types.DatabaseAPI
	identityRegistry *identity.Registry
	spiffeBundles    *spiffe.Bundles
}

// Option configures optional Ingestor dependencies.
type Option func(*ingestor)

// WithIdentityRegistry sets the resolver registry used to verify dns/https/did
// identity and ownership claims. Without it, such claims are indexed as failed
// ("no resolver configured").
func WithIdentityRegistry(r *identity.Registry) Option {
	return func(i *ingestor) { i.identityRegistry = r }
}

// WithSpiffeBundles sets the per-trust-domain CA bundles used to verify SPIFFE
// identity and ownership claims' certificate chains.
func WithSpiffeBundles(b *spiffe.Bundles) Option {
	return func(i *ingestor) { i.spiffeBundles = b }
}

// New creates an Ingestor backed by the given content store and database.
func New(store types.StoreAPI, db types.DatabaseAPI, opts ...Option) Ingestor {
	i := &ingestor{store: store, db: db}

	for _, opt := range opts {
		opt(i)
	}

	return i
}

func (i *ingestor) ImportRecord(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	// Push the record to the content store (source of truth).
	pushedRef, err := i.store.Push(ctx, record)
	if err != nil {
		logger.Error("Failed to push record to store", "error", err)

		return nil, status.Errorf(codes.Internal, "failed to push record to store: %v", err)
	}

	logger.Info("Record pushed to store successfully", "cid", pushedRef.GetCid())

	// Add record to search index for discoverability.
	// Indexing failures are logged but do not fail the import.
	recordData, err := record.Decode()
	if err != nil {
		logger.Error("Failed to decode record for search index", "error", err, "cid", pushedRef.GetCid())

		return pushedRef, nil
	}

	if err := i.db.AddRecord(recordData); err != nil {
		logger.Error("Failed to add record to search index", "error", err, "cid", pushedRef.GetCid())
	} else {
		logger.Debug("Record added to search index successfully", "cid", pushedRef.GetCid())
	}

	return pushedRef, nil
}

func (i *ingestor) ImportReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	// The referrer store handles type-specific storage logic.
	refStore, ok := i.store.(types.ReferrerStoreAPI)
	if !ok {
		return nil, fmt.Errorf("referrer storage not supported by current store implementation")
	}

	referrerRef, err := refStore.PushReferrer(ctx, recordCID, referrer)
	if err != nil {
		return nil, fmt.Errorf("failed to push referrer for record %s: %w", recordCID, err)
	}

	i.applyReferrerDBEffects(ctx, recordCID, referrer)

	logger.Debug("Referrer ingested successfully", "cid", recordCID, "type", referrer.GetType())

	return referrerRef, nil
}

// applyReferrerDBEffects updates referrer-derived database state based on the
// referrer type. All failures are logged but non-fatal: the referrer has
// already been stored, so the content store remains the source of truth.
func (i *ingestor) applyReferrerDBEffects(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) {
	referrerType := referrer.GetType()

	// If this is a signature referrer, mark the record as signed so the name
	// task can find records that need name verification.
	if referrerType == corev1.SignatureReferrerType {
		if err := i.db.SetRecordSigned(recordCID); err != nil {
			logger.Warn("Failed to mark record as signed", "error", err, "cid", recordCID)
		} else {
			logger.Debug("Record marked as signed", "cid", recordCID)
		}
	}

	// Invalidate cached signature verifications when a signature or public key
	// is added so the reconciler re-verifies the record and picks up all signers
	// (e.g. the key signer after a public key is pushed).
	if referrerType == corev1.SignatureReferrerType || referrerType == corev1.PublicKeyReferrerType {
		if err := i.db.InvalidateSignatureVerificationsForRecord(recordCID); err != nil {
			logger.Warn("Failed to invalidate signature verification cache", "error", err, "cid", recordCID)
		} else {
			logger.Debug("Signature verification cache invalidated for record", "cid", recordCID)
		}
	}

	// When a ScanReport referrer is pushed, upsert the scan_reports summary row
	// so the SCANNED and SCAN_SEVERITY search filters reflect the latest result
	// immediately.
	if referrerType == corev1.ScanReportReferrerType {
		report := &securityv1.ScanReport{}
		if err := report.UnmarshalReferrer(&corev1.RecordReferrer{
			Type: referrerType,
			Data: referrer.GetData(),
		}); err != nil {
			logger.Warn("Failed to unmarshal scan report referrer for DB indexing", "error", err, "cid", recordCID)

			return
		}

		if err := i.db.UpsertScanReport(&scanReportRow{
			recordCID:   recordCID,
			scannerType: scannerTypeShortName(report.GetScannerType()),
			isSafe:      report.GetIsSafe(),
			maxSeverity: severityShortName(report.GetMaxSeverity()),
		}, types.DefaultScanSchedule()); err != nil {
			logger.Warn("Failed to upsert scan report summary", "error", err, "cid", recordCID)
		}
	}

	// Eagerly verify and index identity/ownership claims so search reflects a
	// newly pushed claim immediately, without waiting for the reconciler's
	// periodic re-verification pass.
	if referrerType == corev1.IdentityClaimReferrerType {
		i.indexIdentityClaim(ctx, recordCID, referrer)
	}

	if referrerType == corev1.OwnershipClaimReferrerType {
		i.indexOwnershipClaim(ctx, recordCID, referrer)
	}
}

// indexIdentityClaim verifies an identity claim referrer and upserts the result.
func (i *ingestor) indexIdentityClaim(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) {
	var claim identityv1.IdentityClaim
	if err := claim.UnmarshalReferrer(referrer); err != nil {
		logger.Warn("Failed to unmarshal identity claim referrer", "error", err, "cid", recordCID)

		return
	}

	expectedSubject, err := i.declaredSubject(ctx, recordCID, (*corev1.Record).GetIdentity)
	if err != nil {
		logger.Warn("Failed to load record for identity claim verification", "error", err, "cid", recordCID)

		return
	}

	result := identityv1.VerifyIdentityClaim(ctx, &claim, recordCID, expectedSubject, i.resolver(), i.trustedCertsFor(claim.GetSubject()))

	if err := i.db.UpsertClaim(types.ClaimRoleIdentity, claimResultRow{recordCID: recordCID, subject: claim.GetSubject(), result: result}); err != nil {
		logger.Warn("Failed to index identity claim", "error", err, "cid", recordCID)
	} else {
		logger.Debug("Identity claim eagerly indexed", "cid", recordCID, "verified", result.Verified)
	}
}

// indexOwnershipClaim verifies an ownership claim referrer and upserts the result.
func (i *ingestor) indexOwnershipClaim(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) {
	var claim identityv1.OwnershipClaim
	if err := claim.UnmarshalReferrer(referrer); err != nil {
		logger.Warn("Failed to unmarshal ownership claim referrer", "error", err, "cid", recordCID)

		return
	}

	expectedSubject, err := i.declaredSubject(ctx, recordCID, (*corev1.Record).GetOwner)
	if err != nil {
		logger.Warn("Failed to load record for ownership claim verification", "error", err, "cid", recordCID)

		return
	}

	result := identityv1.VerifyOwnershipClaim(ctx, &claim, recordCID, expectedSubject, i.resolver(), i.trustedCertsFor(claim.GetSubject()))

	if err := i.db.UpsertClaim(types.ClaimRoleOwner, claimResultRow{recordCID: recordCID, subject: claim.GetSubject(), result: result}); err != nil {
		logger.Warn("Failed to index ownership claim", "error", err, "cid", recordCID)
	} else {
		logger.Debug("Ownership claim eagerly indexed", "cid", recordCID, "verified", result.Verified)
	}
}

// declaredSubject loads recordCID's content and returns the annotation value
// (identity or owner) that a claim's subject must match, via get.
func (i *ingestor) declaredSubject(ctx context.Context, recordCID string, get func(*corev1.Record) string) (string, error) {
	record, err := i.store.Pull(ctx, &corev1.RecordRef{Cid: recordCID})
	if err != nil {
		return "", fmt.Errorf("pull record %s: %w", recordCID, err)
	}

	return get(record), nil
}

// trustedCertsFor returns the configured SPIFFE trust bundle certificates for
// subject's trust domain, or nil if none are configured.
func (i *ingestor) trustedCertsFor(subject string) []*x509.Certificate {
	if i.spiffeBundles == nil {
		return nil
	}

	return i.spiffeBundles.TrustedCerts(subject)
}

// resolver returns i.identityRegistry as a KeyResolver interface, or a true
// nil interface (not a non-nil interface wrapping a nil pointer) when unset.
func (i *ingestor) resolver() identityv1.KeyResolver {
	if i.identityRegistry == nil {
		return nil
	}

	return i.identityRegistry
}

// claimResultRow adapts an identityv1.Result to types.ClaimObject for DB upsert.
type claimResultRow struct {
	recordCID string
	subject   string
	result    *identityv1.Result
}

func (r claimResultRow) GetRecordCID() string { return r.recordCID }
func (r claimResultRow) GetSubject() string   { return r.subject }

func (r claimResultRow) GetStatus() string {
	if r.result.Verified {
		return "verified"
	}

	return "failed"
}

func (r claimResultRow) GetError() string { return r.result.Error }

func (r claimResultRow) GetVerifiedAt() *time.Time {
	if !r.result.Verified {
		return nil
	}

	now := time.Now()

	return &now
}

// scanReportRow adapts inline scan report data to types.ScanReportObject for DB upsert.
type scanReportRow struct {
	recordCID   string
	scannerType string
	isSafe      bool
	maxSeverity string
}

func (r *scanReportRow) GetRecordCID() string    { return r.recordCID }
func (r *scanReportRow) GetScannerType() string  { return r.scannerType }
func (r *scanReportRow) GetIsSafe() bool         { return r.isSafe }
func (r *scanReportRow) GetMaxSeverity() string  { return r.maxSeverity }
func (r *scanReportRow) GetUpdatedAt() time.Time { return time.Time{} }

// A pushed referrer only ever carries a verdict: the scan report proto has no
// failure fields, since failures are node-local and must not travel to peers.
func (r *scanReportRow) GetStatus() string        { return types.ScanStatusCompleted }
func (r *scanReportRow) GetFailureReason() string { return "" }
func (r *scanReportRow) GetFailureDetail() string { return "" }

// scannerTypeShortName strips the "SCANNER_TYPE_" proto prefix to get the DB column value (e.g. "MCP").
func scannerTypeShortName(t securityv1.ScannerType) string {
	name := t.String()
	if after, ok := strings.CutPrefix(name, "SCANNER_TYPE_"); ok {
		return after
	}

	return name
}

// severityShortName strips the "SEVERITY_" proto prefix to get the DB column value (e.g. "HIGH").
func severityShortName(s securityv1.Severity) string {
	name := s.String()
	if after, ok := strings.CutPrefix(name, "SEVERITY_"); ok {
		return after
	}

	return name
}
