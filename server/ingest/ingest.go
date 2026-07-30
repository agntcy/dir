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
	"fmt"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
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
	store types.StoreAPI
	db    types.DatabaseAPI
}

// New creates an Ingestor backed by the given content store and database.
func New(store types.StoreAPI, db types.DatabaseAPI) Ingestor {
	return &ingestor{store: store, db: db}
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

	i.applyReferrerDBEffects(recordCID, referrer)

	logger.Debug("Referrer ingested successfully", "cid", recordCID, "type", referrer.GetType())

	return referrerRef, nil
}

// applyReferrerDBEffects updates referrer-derived database state based on the
// referrer type. All failures are logged but non-fatal: the referrer has
// already been stored, so the content store remains the source of truth.
func (i *ingestor) applyReferrerDBEffects(recordCID string, referrer *corev1.RecordReferrer) {
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
		}); err != nil {
			logger.Warn("Failed to upsert scan report summary", "error", err, "cid", recordCID)
		}
	}
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
