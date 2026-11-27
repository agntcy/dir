// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"context"
	"errors"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	corev1 "github.com/agntcy/dir/api/core/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/scanner"
)

const otherRecordCID = "baeareiotherrecordcid"

// fakeRecord is a record as GetRecordsNeedingScan returns it. Only the CID is
// read by the task.
type fakeRecord struct {
	coretypes.Record

	cid string
}

func (r fakeRecord) GetCid() string { return r.cid }

// fakeRecordStore serves an empty record for every CID, or pullErr.
type fakeRecordStore struct {
	types.StoreAPI

	pullErr error
}

func (s *fakeRecordStore) Pull(context.Context, *corev1.RecordRef) (*corev1.Record, error) {
	if s.pullErr != nil {
		return nil, s.pullErr
	}

	return &corev1.Record{}, nil
}

// fakeRunner returns a fixed result or error.
type fakeRunner struct {
	name   string
	result *scanner.ScanResult
	err    error
}

func (r *fakeRunner) Name() string { return r.name }

func (r *fakeRunner) Run(context.Context, *corev1.Record) (*scanner.ScanResult, error) {
	return r.result, r.err
}

type startCall struct {
	scanID, recordCID, scannerType, trigger, requestedBy string
}

type finishCall struct {
	id      uint
	version string
	report  types.ScanReportObject
}

type closeCall struct {
	scanID, reason string
}

// fakeScanDB records the scan history and scan_reports writes the task makes.
// Methods the task does not call panic through the nil embedded interface.
type fakeScanDB struct {
	types.DatabaseAPI

	needing []coretypes.Record
	pending []types.SecurityScanObject

	startErr  error
	finishErr error

	nextID   uint
	starts   []startCall
	finishes []finishCall
	discards []uint
	closes   []closeCall
	upserts  []types.ScanReportObject
}

func (f *fakeScanDB) GetRecordsNeedingScan(time.Duration) ([]coretypes.Record, error) {
	return f.needing, nil
}

func (f *fakeScanDB) ListPendingScans(int) ([]types.SecurityScanObject, error) {
	return f.pending, nil
}

func (f *fakeScanDB) StartScannerRun(scanID, recordCID, scannerType, trigger, requestedBy string) (uint, error) {
	if f.startErr != nil {
		return 0, f.startErr
	}

	f.nextID++
	f.starts = append(f.starts, startCall{scanID, recordCID, scannerType, trigger, requestedBy})

	return f.nextID, nil
}

func (f *fakeScanDB) FinishScannerRun(id uint, version string, report types.ScanReportObject, _ types.ScanSchedule) error {
	if f.finishErr != nil {
		return f.finishErr
	}

	f.finishes = append(f.finishes, finishCall{id, version, report})

	return nil
}

func (f *fakeScanDB) DiscardScannerRun(id uint) error {
	f.discards = append(f.discards, id)

	return nil
}

func (f *fakeScanDB) CloseScanPlaceholder(scanID, reason, _ string) error {
	f.closes = append(f.closes, closeCall{scanID, reason})

	return nil
}

func (f *fakeScanDB) UpsertScanReport(report types.ScanReportObject, _ types.ScanSchedule) error {
	f.upserts = append(f.upserts, report)

	return nil
}

// newHistoryTask builds a task with three runners: one that reaches a verdict,
// one that does not apply to the record, and one that errors.
func newHistoryTask(db *fakeScanDB, store *fakeRecordStore) *Task {
	return &Task{
		db:       db,
		store:    store,
		refStore: &fakeReferrerStore{pushedCID: "pushed"},
		runners: []scanner.Runner{
			&fakeRunner{name: "mcp", result: &scanner.ScanResult{Safe: true, Version: "1.2.3"}},
			&fakeRunner{name: "skill", result: &scanner.ScanResult{
				Skipped: true, FailureReason: scanner.FailureArtifactMissing,
			}},
			&fakeRunner{name: "a2a", err: errors.New("scanner exited")},
		},
	}
}

// assertOneScheduledScan checks that want scanner runs started, all scheduled
// and under one shared scan ID.
func assertOneScheduledScan(t *testing.T, starts []startCall, want int) {
	t.Helper()

	if len(starts) != want {
		t.Fatalf("starts = %d, want %d", len(starts), want)
	}

	for _, s := range starts {
		if s.scanID == "" || s.scanID != starts[0].scanID {
			t.Errorf("scan ID %q, want one shared non-empty ID", s.scanID)
		}

		if s.trigger != types.SecurityScanTriggerScheduled {
			t.Errorf("trigger = %q, want scheduled", s.trigger)
		}
	}
}

// A scheduled pass gives every scanner on a record a history row under one
// scan ID, finishes the rows of scanners that ran, discards the one that did
// not apply, and writes scan_reports through the same calls.
func TestRun_ScheduledScanWritesHistory(t *testing.T) {
	t.Parallel()

	db := &fakeScanDB{needing: []coretypes.Record{fakeRecord{cid: testRecordCID}}}

	if err := newHistoryTask(db, &fakeRecordStore{}).Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	assertOneScheduledScan(t, db.starts, 3)

	if len(db.finishes) != 2 {
		t.Fatalf("finishes = %d, want 2 (mcp and a2a)", len(db.finishes))
	}

	if got := db.finishes[0]; got.version != "1.2.3" || got.report.GetStatus() != types.ScanStatusCompleted {
		t.Errorf("mcp finish = version %q status %q, want 1.2.3 completed", got.version, got.report.GetStatus())
	}

	if got := db.finishes[1].report; got.GetScannerType() != "A2A" || got.GetStatus() != types.ScanStatusFailed {
		t.Errorf("a2a finish = %s %s, want A2A failed", got.GetScannerType(), got.GetStatus())
	}

	if len(db.discards) != 1 || db.discards[0] != 2 {
		t.Errorf("discards = %v, want [2] (skill)", db.discards)
	}

	// scan_reports is written inside FinishScannerRun, so nothing is upserted
	// on its own.
	if len(db.upserts) != 0 {
		t.Errorf("upserts = %d, want 0", len(db.upserts))
	}

	if len(db.closes) != 0 {
		t.Errorf("closes = %v, want none for a scheduled scan", db.closes)
	}
}

// History is best effort: when its rows cannot be written, scan_reports is
// still written exactly as before the feature.
func TestRun_HistoryFailureStillWritesScanReports(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		db   *fakeScanDB
	}{
		{"start fails", &fakeScanDB{startErr: errors.New("db down")}},
		{"finish fails", &fakeScanDB{finishErr: errors.New("db down")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := tc.db
			db.needing = []coretypes.Record{fakeRecord{cid: testRecordCID}}

			if err := newHistoryTask(db, &fakeRecordStore{}).Run(t.Context()); err != nil {
				t.Fatalf("Run: %v", err)
			}

			if len(db.upserts) != 2 {
				t.Fatalf("upserts = %d, want 2 (mcp and a2a)", len(db.upserts))
			}

			if db.upserts[0].GetScannerType() != "MCP" || db.upserts[1].GetScannerType() != "A2A" {
				t.Errorf("upserted %s and %s, want MCP and A2A",
					db.upserts[0].GetScannerType(), db.upserts[1].GetScannerType())
			}
		})
	}
}

// A manual scan runs before the scheduled pass, under its own scan ID and
// requester, and the scheduled pass does not scan the same record again.
func TestRun_ManualScanRunsFirstAndOnce(t *testing.T) {
	t.Parallel()

	db := &fakeScanDB{
		pending: []types.SecurityScanObject{&gormdb.SecurityScan{
			ScanID: "manual-1", RecordCID: testRecordCID, RequestedBy: "spiffe://example.org/admin",
		}},
		needing: []coretypes.Record{fakeRecord{cid: testRecordCID}, fakeRecord{cid: otherRecordCID}},
	}

	if err := newHistoryTask(db, &fakeRecordStore{}).Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(db.starts) != 6 {
		t.Fatalf("starts = %d, want 6 (3 scanners x 2 records)", len(db.starts))
	}

	for i, s := range db.starts[:3] {
		if s.recordCID != testRecordCID || s.scanID != "manual-1" ||
			s.trigger != types.SecurityScanTriggerManual || s.requestedBy != "spiffe://example.org/admin" {
			t.Errorf("start %d = %+v, want the manual scan of %s", i, s, testRecordCID)
		}
	}

	for i, s := range db.starts[3:] {
		if s.recordCID != otherRecordCID || s.trigger != types.SecurityScanTriggerScheduled {
			t.Errorf("start %d = %+v, want the scheduled scan of %s", i+3, s, otherRecordCID)
		}
	}

	// Closing is a no-op for a placeholder a scanner took over, so it is
	// always asked for after a manual scan.
	if len(db.closes) != 1 || db.closes[0].scanID != "manual-1" {
		t.Errorf("closes = %v, want one for manual-1", db.closes)
	}
}

// A manual scan whose record cannot be pulled starts no scanner, so its
// placeholder is closed with the reason.
func TestRun_ManualScanPullFailureClosesPlaceholder(t *testing.T) {
	t.Parallel()

	db := &fakeScanDB{pending: []types.SecurityScanObject{&gormdb.SecurityScan{
		ScanID: "manual-1", RecordCID: testRecordCID,
	}}}

	err := newHistoryTask(db, &fakeRecordStore{pullErr: errors.New("not found")}).Run(t.Context())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(db.starts) != 0 {
		t.Errorf("starts = %d, want 0", len(db.starts))
	}

	if len(db.closes) != 1 || db.closes[0].reason == types.SecurityScanFailureNotApplicable {
		t.Errorf("closes = %v, want one with the pull failure's reason", db.closes)
	}
}

// A manual scan no scanner applies to is closed as not applicable.
func TestRun_ManualScanNothingAppliesClosesPlaceholder(t *testing.T) {
	t.Parallel()

	db := &fakeScanDB{pending: []types.SecurityScanObject{&gormdb.SecurityScan{
		ScanID: "manual-1", RecordCID: testRecordCID,
	}}}

	task := newHistoryTask(db, &fakeRecordStore{})
	task.runners = []scanner.Runner{&fakeRunner{name: "skill", result: &scanner.ScanResult{
		Skipped: true, FailureReason: scanner.FailureArtifactMissing,
	}}}

	if err := task.Run(t.Context()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(db.discards) != 1 {
		t.Errorf("discards = %v, want one", db.discards)
	}

	want := closeCall{"manual-1", types.SecurityScanFailureNotApplicable}
	if len(db.closes) != 1 || db.closes[0] != want {
		t.Errorf("closes = %v, want [%v]", db.closes, want)
	}
}
