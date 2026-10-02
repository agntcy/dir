// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"errors"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	dryRunPolicy  = "opa:require-license"
	dryRunVersion = "v7"
)

// verdicts judges a record by its CID: "no licence" for those listed in
// rejected, an error for those in broken, a pass for the rest.
func verdicts(rejected, broken map[string]bool) func(context.Context, coretypes.Record) (bool, string, error) {
	return func(_ context.Context, r coretypes.Record) (bool, string, error) {
		switch {
		case broken[r.GetCid()]:
			return false, "", errors.New("engine failed on " + r.GetCid())
		case rejected[r.GetCid()]:
			return false, "no licence", nil
		default:
			return true, "", nil
		}
	}
}

func dryRunDB(cids ...string) *fakeDB {
	return &fakeDB{records: map[string][]coretypes.Record{dryRunIDPrefix + dryRunPolicy: records(cids...)}}
}

func candidate(evaluate func(context.Context, coretypes.Record) (bool, string, error)) *fakeEvaluator {
	return &fakeEvaluator{id: dryRunPolicy, version: dryRunVersion, evaluate: evaluate}
}

func TestDryRun_ReportsWhatTheCandidateWouldExclude(t *testing.T) {
	t.Parallel()

	db := dryRunDB("cid-a", "cid-b", "cid-c", "cid-d", "cid-e")
	ev := candidate(verdicts(map[string]bool{"cid-b": true, "cid-d": true}, map[string]bool{"cid-c": true}))

	report, err := DryRun(t.Context(), db, ev, Config{}, 10)

	require.NoError(t, err)
	assert.Equal(t, dryRunPolicy, report.PolicyID)
	assert.Equal(t, dryRunVersion, report.PolicyVersion)
	assert.Equal(t, 5, report.Evaluated)
	assert.Equal(t, 2, report.Compliant)
	assert.Equal(t, 2, report.NonCompliant)
	assert.Equal(t, 1, report.Failed)
	assert.Equal(t, 3, report.WouldExclude(), "a record the policy could not judge is excluded too")

	assert.Equal(t, []Excluded{
		{CID: "cid-b", Reason: "no licence"},
		{CID: "cid-c", Reason: "engine failed on cid-c", Failed: true},
		{CID: "cid-d", Reason: "no licence"},
	}, report.Sample, "in CID order, with the reasons")
}

// The point of a dry run: nothing is stored and no version is registered, so
// nothing a read returns can change.
func TestDryRun_StoresNothing(t *testing.T) {
	t.Parallel()

	db := dryRunDB("cid-a", "cid-b")

	_, err := DryRun(t.Context(), db, candidate(verdicts(map[string]bool{"cid-a": true}, nil)), Config{}, 10)

	require.NoError(t, err)
	assert.Empty(t, db.stored, "no verdict is stored")
	assert.Empty(t, db.registered, "no policy version is registered")
}

// Records are selected under an ID no verdict is stored under, so that every
// record is evaluated, not only those the deployed policy has no verdict for.
func TestDryRun_EvaluatesEveryRecordUnderItsOwnID(t *testing.T) {
	t.Parallel()

	db := dryRunDB("cid-a")

	_, err := DryRun(t.Context(), db, candidate(verdicts(nil, nil)), Config{}, 10)

	require.NoError(t, err)
	require.NotEmpty(t, db.selections)

	for _, s := range db.selections {
		assert.Equal(t, "dry-run:"+dryRunPolicy, s.policyID, "never the deployed policy's own ID")
		assert.Equal(t, dryRunVersion, s.version)
	}
}

func TestDryRun_PagesThroughTheRecords(t *testing.T) {
	t.Parallel()

	db := dryRunDB("cid-a", "cid-b", "cid-c", "cid-d", "cid-e")

	report, err := DryRun(t.Context(), db, candidate(verdicts(nil, nil)), Config{BatchSize: 2}, 10)

	require.NoError(t, err)
	assert.Equal(t, 5, report.Evaluated, "every record, whatever the batch size")

	var afters []string
	for _, s := range db.selections {
		assert.Equal(t, 2, s.limit)

		afters = append(afters, s.after)
	}

	assert.Equal(t, []string{"", "cid-b", "cid-d", "cid-e"}, afters, "each batch starts after the last record of the one before")
}

func TestDryRun_KeepsOnlyAsManySamplesAsAskedFor(t *testing.T) {
	t.Parallel()

	rejected := map[string]bool{"cid-a": true, "cid-b": true, "cid-c": true}

	for _, tt := range []struct {
		samples, want int
	}{{0, 0}, {1, 1}, {2, 2}, {3, 3}, {50, 3}} {
		report, err := DryRun(t.Context(), dryRunDB("cid-a", "cid-b", "cid-c"), candidate(verdicts(rejected, nil)), Config{}, tt.samples)

		require.NoError(t, err)
		assert.Len(t, report.Sample, tt.want, "samples=%d", tt.samples)
		assert.Equal(t, 3, report.NonCompliant, "the count does not depend on the sample")
	}
}

func TestDryRun_NothingToEvaluate(t *testing.T) {
	t.Parallel()

	report, err := DryRun(t.Context(), dryRunDB(), candidate(verdicts(nil, nil)), Config{}, 10)

	require.NoError(t, err)
	assert.Zero(t, report.Evaluated)
	assert.Zero(t, report.WouldExclude())
	assert.NotNil(t, report.Sample, "an empty sample is a list, not null, in the output")
	assert.Empty(t, report.Sample)
}

// A record that takes too long fails the dry run as it would fail the task.
func TestDryRun_AppliesTheRecordTimeout(t *testing.T) {
	t.Parallel()

	ev := candidate(func(ctx context.Context, _ coretypes.Record) (bool, string, error) {
		<-ctx.Done()

		return false, "", ctx.Err()
	})

	report, err := DryRun(t.Context(), dryRunDB("cid-a"), ev, Config{RecordTimeout: 10 * time.Millisecond}, 10)

	require.NoError(t, err, "a record's timeout is not the dry run's")
	assert.Equal(t, 1, report.Failed)
	assert.Contains(t, report.Sample[0].Reason, "deadline exceeded")
}

// When the dry run itself is interrupted, the record in hand says nothing about
// the policy and is not counted.
func TestDryRun_InterruptionReturnsTheReportSoFar(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ev := candidate(func(_ context.Context, r coretypes.Record) (bool, string, error) {
		if r.GetCid() == "cid-c" {
			cancel()

			return false, "", context.Canceled
		}

		return true, "", nil
	})

	report, err := DryRun(ctx, dryRunDB("cid-a", "cid-b", "cid-c", "cid-d"), ev, Config{}, 10)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 2, report.Evaluated, "the interrupted record is not counted, nor are those after it")
	assert.Zero(t, report.Failed)
}

func TestDryRun_SelectionFailure(t *testing.T) {
	t.Parallel()

	db := dryRunDB("cid-a")
	db.selectErr = map[string]error{dryRunIDPrefix + dryRunPolicy: errors.New("database unavailable")}

	report, err := DryRun(t.Context(), db, candidate(verdicts(nil, nil)), Config{}, 10)

	require.ErrorContains(t, err, "database unavailable")
	assert.Zero(t, report.Evaluated)
}
