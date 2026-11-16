// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Every error is a gRPC status for the caller, returned as is.
//
//nolint:wrapcheck
package controller

import (
	"context"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/authn"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var scanLogger = logging.Logger("controller/scan")

// ScanDatabase is what the scan service reads and writes: whether a record is
// indexed, and its security scan history.
type ScanDatabase interface {
	GetRecordIndexedAt(cid string) (time.Time, error)
	CreatePendingScan(recordCID, requestedBy string) (types.SecurityScanObject, bool, error)
	ListScans(recordCID, status string, offset, limit int) ([]types.SecurityScanObject, error)
}

type scanCtlr struct {
	db ScanDatabase
}

// NewScanController serves manual security scan requests and scan history.
// The reconciler runs the scans; this only queues them and reads them back.
func NewScanController(db ScanDatabase) securityv1.ScanServiceServer {
	return &scanCtlr{db: db}
}

func (c *scanCtlr) NewScan(ctx context.Context, req *securityv1.NewScanRequest) (*securityv1.NewScanResponse, error) {
	cid := req.GetCid()
	if err := validateScanCID(cid); err != nil {
		return nil, err
	}

	// A scan is stored against the record's row, so only an indexed record
	// can be scanned.
	indexedAt, err := c.db.GetRecordIndexedAt(cid)
	if err != nil {
		return nil, internalScanError("look up record", err)
	}

	if indexedAt.IsZero() {
		return nil, status.Errorf(codes.NotFound, "record %q is not indexed on this node", cid)
	}

	caller := callerID(ctx)

	scan, created, err := c.db.CreatePendingScan(cid, caller)
	if err != nil {
		return nil, internalScanError("create scan", err)
	}

	scanLogger.Info("Security scan requested", "cid", cid, "caller", caller,
		"scan_id", scan.GetScanID(), "created", created)

	return &securityv1.NewScanResponse{Scan: toProtoScan(scan), Created: created}, nil
}

func (c *scanCtlr) ListScans(req *securityv1.ListScansRequest, srv securityv1.ScanService_ListScansServer) error {
	cid := req.GetCid()
	if err := validateScanCID(cid); err != nil {
		return err
	}

	statusFilter, ok := fromProtoScanStatus(req.GetStatus())
	if !ok {
		return status.Errorf(codes.InvalidArgument, "unknown scan status %v", req.GetStatus())
	}

	scans, err := c.db.ListScans(cid, statusFilter, int(req.GetOffset()), int(req.GetLimit()))
	if err != nil {
		return internalScanError("list scans", err)
	}

	for _, scan := range scans {
		if err := srv.Send(&securityv1.ListScansResponse{Scan: toProtoScan(scan)}); err != nil {
			return err
		}
	}

	return nil
}

func validateScanCID(cid string) error {
	if _, err := corev1.ConvertCIDToDigest(cid); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid record CID %q", cid)
	}

	return nil
}

// callerID is the caller's SPIFFE ID, or empty without authentication.
func callerID(ctx context.Context) string {
	if id, ok := authn.SpiffeIDFromContext(ctx); ok {
		return id.String()
	}

	return ""
}

// internalScanError logs err and answers Internal without its detail.
func internalScanError(what string, err error) error {
	scanLogger.Error("Scan service: failed to "+what, "error", err)

	return status.Error(codes.Internal, "failed to "+what)
}

func toProtoScan(s types.SecurityScanObject) *securityv1.Scan {
	scan := &securityv1.Scan{
		ScanId:         s.GetScanID(),
		Cid:            s.GetRecordCID(),
		ScannerType:    securityv1.ScannerType(securityv1.ScannerType_value["SCANNER_TYPE_"+s.GetScannerType()]),
		ScannerVersion: s.GetScannerVersion(),
		Trigger:        toProtoScanTrigger(s.GetTrigger()),
		Status:         toProtoScanStatus(s.GetStatus()),
		IsSafe:         s.GetIsSafe(),
		FindingCount:   uint32(max(s.GetFindingCount(), 0)), //nolint:gosec // clamped to zero, and a count never nears 2^32.
		Partial:        s.GetPartial(),
		FailureReason:  s.GetFailureReason(),
		FailureDetail:  s.GetFailureDetail(),
		RequestedBy:    s.GetRequestedBy(),
		CreatedAt:      optionalTimestamp(s.GetCreatedAt()),
		StartedAt:      optionalTimestamp(s.GetStartedAt()),
		FinishedAt:     optionalTimestamp(s.GetFinishedAt()),
	}

	// Set once a scanner has reached a verdict; unspecified until then.
	if s.GetMaxSeverity() != "" {
		scan.MaxSeverity = securityv1.Severity(securityv1.Severity_value["SEVERITY_"+s.GetMaxSeverity()])
	}

	return scan
}

// optionalTimestamp leaves unset times unset rather than sending 1970.
func optionalTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}

	return timestamppb.New(t)
}

func toProtoScanStatus(s string) securityv1.ScanStatus {
	switch s {
	case types.SecurityScanStatusPending:
		return securityv1.ScanStatus_SCAN_STATUS_PENDING
	case types.SecurityScanStatusRunning:
		return securityv1.ScanStatus_SCAN_STATUS_RUNNING
	case types.SecurityScanStatusSucceeded:
		return securityv1.ScanStatus_SCAN_STATUS_SUCCEEDED
	case types.SecurityScanStatusFailed:
		return securityv1.ScanStatus_SCAN_STATUS_FAILED
	default:
		return securityv1.ScanStatus_SCAN_STATUS_UNSPECIFIED
	}
}

// fromProtoScanStatus maps a requested status filter onto the stored value.
// Unspecified means no filter, and an unknown value is not ok.
func fromProtoScanStatus(s securityv1.ScanStatus) (string, bool) {
	switch s {
	case securityv1.ScanStatus_SCAN_STATUS_UNSPECIFIED:
		return "", true
	case securityv1.ScanStatus_SCAN_STATUS_PENDING:
		return types.SecurityScanStatusPending, true
	case securityv1.ScanStatus_SCAN_STATUS_RUNNING:
		return types.SecurityScanStatusRunning, true
	case securityv1.ScanStatus_SCAN_STATUS_SUCCEEDED:
		return types.SecurityScanStatusSucceeded, true
	case securityv1.ScanStatus_SCAN_STATUS_FAILED:
		return types.SecurityScanStatusFailed, true
	default:
		return "", false
	}
}

func toProtoScanTrigger(s string) securityv1.ScanTrigger {
	switch s {
	case types.SecurityScanTriggerScheduled:
		return securityv1.ScanTrigger_SCAN_TRIGGER_SCHEDULED
	case types.SecurityScanTriggerManual:
		return securityv1.ScanTrigger_SCAN_TRIGGER_MANUAL
	default:
		return securityv1.ScanTrigger_SCAN_TRIGGER_UNSPECIFIED
	}
}
