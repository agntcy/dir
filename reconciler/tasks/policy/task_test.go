// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"errors"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRecord implements coretypes.Record for tests; the task only reads GetCid.
type fakeRecord struct {
	coretypes.Record

	cid string
}

func (r *fakeRecord) GetCid() string { return r.cid }

// fakeDB implements types.DatabaseAPI via embedding; only the methods the
// task uses are overridden, so a call to anything else panics and surfaces.
type fakeDB struct {
	types.DatabaseAPI

	records   map[string][]coretypes.Record // keyed by policy ID
	selectErr map[string]error              // keyed by policy ID
	upsertErr func(*gormdb.PolicyEvaluation) error

	selectedVersions map[string]string
	stored           []*gormdb.PolicyEvaluation
}

func (f *fakeDB) GetRecordsNeedingPolicyEvaluation(policyID, policyVersion string) ([]coretypes.Record, error) {
	if f.selectedVersions == nil {
		f.selectedVersions = make(map[string]string)
	}

	f.selectedVersions[policyID] = policyVersion

	if err := f.selectErr[policyID]; err != nil {
		return nil, err
	}

	return f.records[policyID], nil
}

func (f *fakeDB) UpsertPolicyEvaluation(eval types.PolicyEvaluationObject) error {
	row, ok := eval.(*gormdb.PolicyEvaluation)
	if !ok {
		return errors.New("unexpected policy evaluation type")
	}

	if f.upsertErr != nil {
		if err := f.upsertErr(row); err != nil {
			return err
		}
	}

	f.stored = append(f.stored, row)

	return nil
}

// fakeEvaluator is an Evaluator whose verdicts come from evaluate. version is
// a field rather than a constant so a test can reload the policy mid-run.
type fakeEvaluator struct {
	id       string
	version  string
	calls    int
	evaluate func(context.Context, coretypes.Record) (bool, string, error)
}

func (e *fakeEvaluator) PolicyID() string      { return e.id }
func (e *fakeEvaluator) PolicyVersion() string { return e.version }

func (e *fakeEvaluator) Evaluate(ctx context.Context, record coretypes.Record) (bool, string, error) {
	e.calls++

	return e.evaluate(ctx, record)
}

func records(cids ...string) []coretypes.Record {
	out := make([]coretypes.Record, 0, len(cids))
	for _, cid := range cids {
		out = append(out, &fakeRecord{cid: cid})
	}

	return out
}

func storedFor(t *testing.T, db *fakeDB, cid string) *gormdb.PolicyEvaluation {
	t.Helper()

	for _, row := range db.stored {
		if row.RecordCID == cid {
			return row
		}
	}

	require.FailNow(t, "no evaluation stored", "record %s", cid)

	return nil
}

// --- NewTask ---

func TestNewTask_RejectsInvalidEvaluators(t *testing.T) {
	t.Parallel()

	valid := &fakeEvaluator{id: "opa:a", version: "v1"}

	tests := []struct {
		name       string
		evaluators []Evaluator
		wantErr    string
	}{
		{"nil evaluator", []Evaluator{nil}, "evaluator 0 is nil"},
		{"empty policy ID", []Evaluator{&fakeEvaluator{version: "v1"}}, "empty policy ID"},
		{"empty policy version", []Evaluator{&fakeEvaluator{id: "opa:a"}}, "empty version"},
		// Two evaluators under one policy ID would overwrite each other's
		// rows, flapping the verdict every run.
		{"duplicate policy ID", []Evaluator{valid, &fakeEvaluator{id: "opa:a", version: "v2"}}, "registered more than once"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewTask(Config{}, &fakeDB{}, tt.evaluators...)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// --- IsEnabled ---

func TestIsEnabled(t *testing.T) {
	t.Parallel()

	ev := &fakeEvaluator{id: "opa:a", version: "v1"}

	tests := []struct {
		name       string
		enabled    bool
		evaluators []Evaluator
		want       bool
	}{
		{"disabled in configuration", false, []Evaluator{ev}, false},
		// Nothing to evaluate: scheduling it would only log empty runs.
		{"enabled with no policies", true, nil, false},
		{"enabled with a policy", true, []Evaluator{ev}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			task, err := NewTask(Config{Enabled: tt.enabled}, &fakeDB{}, tt.evaluators...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, task.IsEnabled())
		})
	}
}

// --- Run ---

func TestRun_StoresVerdicts(t *testing.T) {
	t.Parallel()

	db := &fakeDB{records: map[string][]coretypes.Record{"opa:a": records("cid-ok", "cid-bad")}}
	ev := &fakeEvaluator{id: "opa:a", version: "v1", evaluate: func(_ context.Context, r coretypes.Record) (bool, string, error) {
		if r.GetCid() == "cid-ok" {
			// A reason on a pass is dropped: the column is empty when compliant.
			return true, "all annotations present", nil
		}

		return false, "missing annotation", nil
	}}

	task, err := NewTask(Config{Enabled: true}, db, ev)
	require.NoError(t, err)
	require.NoError(t, task.Run(t.Context()))

	require.Len(t, db.stored, 2)

	ok := storedFor(t, db, "cid-ok")
	assert.True(t, ok.Compliant)
	assert.Equal(t, types.PolicyEvalStatusEvaluated, ok.Status)
	assert.Empty(t, ok.Reason)
	assert.Equal(t, "opa:a", ok.PolicyID)
	assert.Equal(t, "v1", ok.PolicyVersion)

	bad := storedFor(t, db, "cid-bad")
	assert.False(t, bad.Compliant)
	assert.Equal(t, types.PolicyEvalStatusEvaluated, bad.Status)
	assert.Equal(t, "missing annotation", bad.Reason)
}

// The fail-closed rule: an evaluator error is stored as a failed evaluation,
// which reads as non-compliant, never as a pass — and the error's own text,
// which may name internal hosts or paths, stays out of the queryable row.
func TestRun_EvaluatorErrorIsAFailedVerdictWithoutItsDetail(t *testing.T) {
	t.Parallel()

	db := &fakeDB{records: map[string][]coretypes.Record{"opa:a": records("cid-a")}}
	ev := &fakeEvaluator{id: "opa:a", version: "v1", evaluate: func(context.Context, coretypes.Record) (bool, string, error) {
		// compliant=true alongside an error, to prove the error wins.
		return true, "", errors.New("dial tcp 10.0.0.5:8181: connection refused")
	}}

	task, err := NewTask(Config{Enabled: true}, db, ev)
	require.NoError(t, err)

	// A per-record failure is a verdict, not a task failure.
	require.NoError(t, task.Run(t.Context()))

	row := storedFor(t, db, "cid-a")
	assert.False(t, row.Compliant)
	assert.Equal(t, types.PolicyEvalStatusFailed, row.Status)
	assert.Equal(t, failedReason, row.Reason)
	assert.NotContains(t, row.Reason, "10.0.0.5")
}

// A slow evaluator is a failed evaluation of that record, not a shutdown: the
// run's own context is still live, so the failure is stored and retried.
func TestRun_RecordTimeoutIsAFailedVerdict(t *testing.T) {
	t.Parallel()

	db := &fakeDB{records: map[string][]coretypes.Record{"opa:a": records("cid-a")}}
	ev := &fakeEvaluator{id: "opa:a", version: "v1", evaluate: func(ctx context.Context, _ coretypes.Record) (bool, string, error) {
		<-ctx.Done()

		return false, "", ctx.Err()
	}}

	task, err := NewTask(Config{Enabled: true, RecordTimeout: time.Millisecond}, db, ev)
	require.NoError(t, err)
	require.NoError(t, task.Run(t.Context()))

	row := storedFor(t, db, "cid-a")
	assert.Equal(t, types.PolicyEvalStatusFailed, row.Status)
	assert.False(t, row.Compliant)
}

// On shutdown the evaluator reports the cancellation, which says nothing
// about the record. Storing it as a failed evaluation would mark every
// remaining record non-compliant on every restart.
func TestRun_ShutdownStoresNothing(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	db := &fakeDB{records: map[string][]coretypes.Record{"opa:a": records("cid-a", "cid-b")}}
	ev := &fakeEvaluator{id: "opa:a", version: "v1", evaluate: func(recordCtx context.Context, _ coretypes.Record) (bool, string, error) {
		cancel()

		return false, "", recordCtx.Err()
	}}

	task, err := NewTask(Config{Enabled: true}, db, ev)
	require.NoError(t, err)

	err = task.Run(ctx)
	require.ErrorIs(t, err, context.Canceled)

	assert.Empty(t, db.stored)
	assert.Equal(t, 1, ev.calls, "the second record must not be evaluated after shutdown")
}

func TestRun_SelectionErrorDoesNotStopOtherPolicies(t *testing.T) {
	t.Parallel()

	selectErr := errors.New("connection reset")
	db := &fakeDB{
		selectErr: map[string]error{"opa:a": selectErr},
		records:   map[string][]coretypes.Record{"opa:b": records("cid-x")},
	}
	pass := func(context.Context, coretypes.Record) (bool, string, error) { return true, "", nil }

	task, err := NewTask(Config{Enabled: true}, db,
		&fakeEvaluator{id: "opa:a", version: "v1", evaluate: pass},
		&fakeEvaluator{id: "opa:b", version: "v1", evaluate: pass},
	)
	require.NoError(t, err)

	err = task.Run(t.Context())
	require.ErrorIs(t, err, selectErr)
	require.ErrorContains(t, err, `"opa:a"`)

	row := storedFor(t, db, "cid-x")
	assert.Equal(t, "opa:b", row.PolicyID)
}

func TestRun_StoreErrorDoesNotStopTheRun(t *testing.T) {
	t.Parallel()

	db := &fakeDB{
		records: map[string][]coretypes.Record{"opa:a": records("cid-a", "cid-b")},
		upsertErr: func(row *gormdb.PolicyEvaluation) error {
			if row.RecordCID == "cid-a" {
				return errors.New("disk full")
			}

			return nil
		},
	}
	ev := &fakeEvaluator{id: "opa:a", version: "v1", evaluate: func(context.Context, coretypes.Record) (bool, string, error) {
		return true, "", nil
	}}

	task, err := NewTask(Config{Enabled: true}, db, ev)
	require.NoError(t, err)
	require.NoError(t, task.Run(t.Context()))

	require.Len(t, db.stored, 1)
	assert.Equal(t, "cid-b", db.stored[0].RecordCID)
}

// Rows carry the version the records were selected under. If the evaluator
// reloads mid-run, those rows are stale on the next run and re-selected under
// the new version — rather than labelled with a version they were never
// selected for.
func TestRun_RowsCarryTheVersionTheyWereSelectedUnder(t *testing.T) {
	t.Parallel()

	db := &fakeDB{records: map[string][]coretypes.Record{"opa:a": records("cid-a", "cid-b")}}
	ev := &fakeEvaluator{id: "opa:a", version: "v1"}
	ev.evaluate = func(context.Context, coretypes.Record) (bool, string, error) {
		ev.version = "v2"

		return true, "", nil
	}

	task, err := NewTask(Config{Enabled: true}, db, ev)
	require.NoError(t, err)
	require.NoError(t, task.Run(t.Context()))

	assert.Equal(t, "v1", db.selectedVersions["opa:a"])

	require.Len(t, db.stored, 2)

	for _, row := range db.stored {
		assert.Equal(t, "v1", row.PolicyVersion)
	}
}

func TestRun_NoRecordsNeedEvaluation(t *testing.T) {
	t.Parallel()

	db := &fakeDB{}
	ev := &fakeEvaluator{id: "opa:a", version: "v1", evaluate: func(context.Context, coretypes.Record) (bool, string, error) {
		return true, "", nil
	}}

	task, err := NewTask(Config{Enabled: true}, db, ev)
	require.NoError(t, err)
	require.NoError(t, task.Run(t.Context()))

	assert.Empty(t, db.stored)
	assert.Zero(t, ev.calls)
}
