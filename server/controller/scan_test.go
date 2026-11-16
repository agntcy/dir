// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/authn"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const scanAdmin = "spiffe://example.org/ns/dir/sa/admin"

// scanDB serves one indexed record, gatedCID, and records what the
// controller asks of it.
type scanDB struct {
	scans   []types.SecurityScanObject
	created bool
	err     error

	requestedBy  string
	listStatus   string
	listOffset   int
	listLimit    int
	createCalled bool
}

func (d *scanDB) GetRecordIndexedAt(cid string) (time.Time, error) {
	if cid != gatedCID {
		return time.Time{}, nil
	}

	return time.Unix(1700000000, 0), nil
}

func (d *scanDB) CreatePendingScan(_, requestedBy string) (types.SecurityScanObject, bool, error) {
	d.createCalled = true
	d.requestedBy = requestedBy

	if d.err != nil {
		return nil, false, d.err
	}

	return d.scans[0], d.created, nil
}

func (d *scanDB) ListScans(_, status string, offset, limit int) ([]types.SecurityScanObject, error) {
	d.listStatus, d.listOffset, d.listLimit = status, offset, limit

	return d.scans, d.err
}

type scanStream struct {
	securityv1.ScanService_ListScansServer

	ctx  context.Context //nolint:containedctx // a gRPC stream carries its context
	sent []*securityv1.Scan
}

func (s *scanStream) Context() context.Context { return s.ctx }

func (s *scanStream) Send(resp *securityv1.ListScansResponse) error {
	s.sent = append(s.sent, resp.GetScan())

	return nil
}

func asScanAdmin(t *testing.T) context.Context {
	t.Helper()

	return context.WithValue(t.Context(), authn.SpiffeIDContextKey, spiffeid.RequireFromString(scanAdmin))
}

func pendingScan() *gormdb.SecurityScan {
	return &gormdb.SecurityScan{
		ScanID:      "scan-1",
		RecordCID:   gatedCID,
		Trigger:     types.SecurityScanTriggerManual,
		Status:      types.SecurityScanStatusPending,
		RequestedBy: scanAdmin,
		CreatedAt:   time.Unix(1700000000, 0),
	}
}

func succeededScan() *gormdb.SecurityScan {
	started, finished := time.Unix(1700000100, 0), time.Unix(1700000200, 0)

	return &gormdb.SecurityScan{
		ScanID:         "scan-2",
		RecordCID:      gatedCID,
		ScannerType:    "MCP",
		ScannerVersion: "1.2.3",
		Trigger:        types.SecurityScanTriggerScheduled,
		Status:         types.SecurityScanStatusSucceeded,
		MaxSeverity:    "HIGH",
		FindingCount:   2,
		CreatedAt:      started,
		StartedAt:      &started,
		FinishedAt:     &finished,
	}
}

// A new scan is queued under the caller's identity and returned pending.
func TestScan_NewScanQueuesUnderCaller(t *testing.T) {
	t.Parallel()

	db := &scanDB{scans: []types.SecurityScanObject{pendingScan()}, created: true}

	resp, err := NewScanController(db).NewScan(asScanAdmin(t), &securityv1.NewScanRequest{Cid: gatedCID})
	require.NoError(t, err)

	assert.Equal(t, scanAdmin, db.requestedBy)
	assert.True(t, resp.GetCreated())

	scan := resp.GetScan()
	assert.Equal(t, "scan-1", scan.GetScanId())
	assert.Equal(t, securityv1.ScanStatus_SCAN_STATUS_PENDING, scan.GetStatus())
	assert.Equal(t, securityv1.ScanTrigger_SCAN_TRIGGER_MANUAL, scan.GetTrigger())
	assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_UNSPECIFIED, scan.GetScannerType())
	assert.Equal(t, securityv1.Severity_SEVERITY_UNSPECIFIED, scan.GetMaxSeverity())
	assert.NotNil(t, scan.GetCreatedAt())
	assert.Nil(t, scan.GetStartedAt(), "no scanner has started")
	assert.Nil(t, scan.GetFinishedAt())
}

// Asking again while a scan is pending returns that scan, not a new one.
func TestScan_NewScanReturnsExisting(t *testing.T) {
	t.Parallel()

	db := &scanDB{scans: []types.SecurityScanObject{pendingScan()}, created: false}

	resp, err := NewScanController(db).NewScan(asScanAdmin(t), &securityv1.NewScanRequest{Cid: gatedCID})
	require.NoError(t, err)
	assert.False(t, resp.GetCreated())
	assert.Equal(t, "scan-1", resp.GetScan().GetScanId())
}

func TestScan_NewScanFailures(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		cid  string
		err  error
		want codes.Code
	}{
		{"empty CID", "", nil, codes.InvalidArgument},
		{"malformed CID", "not-a-cid", nil, codes.InvalidArgument},
		{"record not indexed", testCID, nil, codes.NotFound},
		{"database error", gatedCID, errors.New("connection refused"), codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := &scanDB{scans: []types.SecurityScanObject{pendingScan()}, err: tc.err}

			_, err := NewScanController(db).NewScan(asScanAdmin(t), &securityv1.NewScanRequest{Cid: tc.cid})
			assert.Equal(t, tc.want, status.Code(err))

			if tc.want != codes.Internal {
				assert.False(t, db.createCalled, "nothing is queued for a bad request")
			} else {
				assert.NotContains(t, err.Error(), "connection refused", "internal detail is not returned")
			}
		})
	}
}

// ListScans streams each scan with its fields mapped, and passes the filter
// and paging through.
func TestScan_ListScans(t *testing.T) {
	t.Parallel()

	db := &scanDB{scans: []types.SecurityScanObject{succeededScan(), pendingScan()}}
	stream := &scanStream{ctx: asScanAdmin(t)}

	err := NewScanController(db).ListScans(&securityv1.ListScansRequest{
		Cid:    gatedCID,
		Status: securityv1.ScanStatus_SCAN_STATUS_SUCCEEDED,
		Limit:  10,
		Offset: 5,
	}, stream)
	require.NoError(t, err)

	assert.Equal(t, types.SecurityScanStatusSucceeded, db.listStatus)
	assert.Equal(t, 5, db.listOffset)
	assert.Equal(t, 10, db.listLimit)

	require.Len(t, stream.sent, 2)

	scan := stream.sent[0]
	assert.Equal(t, "scan-2", scan.GetScanId())
	assert.Equal(t, securityv1.ScannerType_SCANNER_TYPE_MCP, scan.GetScannerType())
	assert.Equal(t, "1.2.3", scan.GetScannerVersion())
	assert.Equal(t, securityv1.ScanTrigger_SCAN_TRIGGER_SCHEDULED, scan.GetTrigger())
	assert.Equal(t, securityv1.ScanStatus_SCAN_STATUS_SUCCEEDED, scan.GetStatus())
	assert.Equal(t, securityv1.Severity_SEVERITY_HIGH, scan.GetMaxSeverity())
	assert.Equal(t, uint32(2), scan.GetFindingCount())
	assert.Equal(t, int64(1700000100), scan.GetStartedAt().GetSeconds())
	assert.Equal(t, int64(1700000200), scan.GetFinishedAt().GetSeconds())
}

func TestScan_ListScansUnspecifiedStatusMeansAll(t *testing.T) {
	t.Parallel()

	db := &scanDB{}

	err := NewScanController(db).ListScans(&securityv1.ListScansRequest{Cid: gatedCID}, &scanStream{ctx: asScanAdmin(t)})
	require.NoError(t, err)
	assert.Empty(t, db.listStatus)
}

func TestScan_ListScansFailures(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		req  *securityv1.ListScansRequest
		err  error
		want codes.Code
	}{
		{"malformed CID", &securityv1.ListScansRequest{Cid: "not-a-cid"}, nil, codes.InvalidArgument},
		{"unknown status", &securityv1.ListScansRequest{Cid: gatedCID, Status: 99}, nil, codes.InvalidArgument},
		{"database error", &securityv1.ListScansRequest{Cid: gatedCID}, errors.New("connection refused"), codes.Internal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := &scanDB{err: tc.err}

			err := NewScanController(db).ListScans(tc.req, &scanStream{ctx: asScanAdmin(t)})
			assert.Equal(t, tc.want, status.Code(err))
		})
	}
}
