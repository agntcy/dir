// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"context"
	"errors"
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- merge ---

func TestMerge_Empty(t *testing.T) {
	t.Parallel()

	got := merge(nil)
	if got == nil || !got.Skipped || got.SkippedReason == "" {
		t.Errorf("empty input must produce a skipped result: %+v", got)
	}
}

func TestMerge_SinglePassThrough(t *testing.T) {
	t.Parallel()

	in := &ScanResult{Safe: true}
	got := merge([]*ScanResult{in})

	if got != in {
		t.Error("single result should be returned as-is")
	}
}

func TestMerge_AllSafe(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{{Safe: true}, {Safe: true}})
	if !got.Safe || got.Skipped {
		t.Errorf("all-safe should be Safe and not Skipped: %+v", got)
	}
}

func TestMerge_OneNotSafeWinsOverall(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{
		{Safe: true},
		{Safe: false, Findings: []Finding{{Severity: SeverityError, Message: "boom"}}},
	})

	if got.Safe {
		t.Error("any non-safe result should make the merged result non-safe")
	}

	if len(got.Findings) != 1 {
		t.Errorf("findings should be merged, got %d", len(got.Findings))
	}
}

func TestMerge_AllSkipped(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{
		{Skipped: true, SkippedReason: "no rules"},
		{Skipped: true, SkippedReason: "no runners"},
	})

	if !got.Skipped {
		t.Error("all skipped → merged should be skipped")
	}

	if got.Safe {
		t.Error("skipped merged result must not be marked safe")
	}

	if got.SkippedReason == "" {
		t.Error("merged skip reason should be populated")
	}
}

func TestMerge_NilElementsSkipped(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{nil, {Safe: true}, nil})
	if !got.Safe {
		t.Error("nil elements should be skipped; surviving safe result should win")
	}
}

func TestMerge_FindingsAlwaysMakeSafeFalse(t *testing.T) {
	t.Parallel()

	got := merge([]*ScanResult{
		{Safe: true, Findings: []Finding{{Severity: SeverityInfo, Message: "fyi"}}},
		{Safe: true},
	})

	if got.Safe {
		t.Error("merged result with any finding should be unsafe")
	}
}

// --- HasError / HasWarning ---

func TestHasError_False(t *testing.T) {
	t.Parallel()

	r := &ScanResult{Findings: []Finding{{Severity: SeverityWarning}}}
	if r.HasError() {
		t.Error("no error-severity finding should return false")
	}
}

func TestHasError_True(t *testing.T) {
	t.Parallel()

	r := &ScanResult{Findings: []Finding{{Severity: SeverityError}}}
	if !r.HasError() {
		t.Error("error-severity finding should return true")
	}
}

func TestHasWarning_False(t *testing.T) {
	t.Parallel()

	r := &ScanResult{Findings: []Finding{{Severity: SeverityInfo}}}
	if r.HasWarning() {
		t.Error("no warning-severity finding should return false")
	}
}

func TestHasWarning_True(t *testing.T) {
	t.Parallel()

	r := &ScanResult{Findings: []Finding{{Severity: SeverityWarning}}}
	if !r.HasWarning() {
		t.Error("warning-severity finding should return true")
	}
}

// --- RunAll ---

type stubRunner struct {
	name   string
	result *ScanResult
	err    error
}

func (s *stubRunner) Name() string { return s.name }
func (s *stubRunner) Run(_ context.Context, _ *corev1.Record) (*ScanResult, error) {
	return s.result, s.err
}

func newRecord(t *testing.T) *corev1.Record {
	t.Helper()

	st, err := structpb.NewStruct(map[string]any{"name": "test", "version": "1.0.0"})
	if err != nil {
		t.Fatalf("structpb.NewStruct: %v", err)
	}

	return &corev1.Record{Data: st}
}

func TestRunAll_NoRunners_ReturnsSkipped(t *testing.T) {
	t.Parallel()

	got, err := RunAll(context.Background(), nil, newRecord(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Skipped {
		t.Error("no runners should produce a skipped result")
	}
}

func TestRunAll_AllSucceed_MergesResults(t *testing.T) {
	t.Parallel()

	runners := []Runner{
		&stubRunner{name: "a", result: &ScanResult{Safe: true}},
		&stubRunner{name: "b", result: &ScanResult{Safe: true}},
	}

	got, err := RunAll(context.Background(), runners, newRecord(t), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !got.Safe {
		t.Error("all-safe runners should produce a safe result")
	}
}

func TestRunAll_AllFail_ReturnsError(t *testing.T) {
	t.Parallel()

	runners := []Runner{
		&stubRunner{name: "a", err: errors.New("offline")},
		&stubRunner{name: "b", err: errors.New("offline")},
	}

	_, err := RunAll(context.Background(), runners, newRecord(t), nil)
	if err == nil {
		t.Error("all runners failing should return an error")
	}
}

type captureLogger struct{ msgs []string }

func (l *captureLogger) Warn(msg string, _ ...any) { l.msgs = append(l.msgs, msg) }

func TestRunAll_FailedRunnerLogsViaLogger(t *testing.T) {
	t.Parallel()

	log := &captureLogger{}

	runners := []Runner{
		&stubRunner{name: "down", err: errors.New("offline")},
		&stubRunner{name: "up", result: &ScanResult{Safe: true}},
	}

	_, err := RunAll(context.Background(), runners, newRecord(t), log)
	if err != nil {
		t.Fatalf("partial failure should not error: %v", err)
	}

	if len(log.msgs) == 0 {
		t.Error("expected at least one Warn call for the failing runner")
	}
}

func TestRunAll_OneFailsOtherSucceeds_NoError(t *testing.T) {
	t.Parallel()

	runners := []Runner{
		&stubRunner{name: "ok", result: &ScanResult{Safe: true}},
		&stubRunner{name: "bad", err: errors.New("offline")},
	}

	got, err := RunAll(context.Background(), runners, newRecord(t), nil)
	if err != nil {
		t.Fatalf("partial failure should not return an error: %v", err)
	}

	if !got.Safe {
		t.Error("surviving runner's safe result should win")
	}
}
