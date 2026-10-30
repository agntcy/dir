// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// storedVerdict is one row as the database would hand it over.
type storedVerdict struct {
	policyID  string
	version   string
	status    string
	compliant bool
	reason    string
	updatedAt time.Time
}

func (v storedVerdict) GetRecordCID() string     { return verdictTestCID }
func (v storedVerdict) GetPolicyID() string      { return v.policyID }
func (v storedVerdict) GetPolicyVersion() string { return v.version }
func (v storedVerdict) GetCompliant() bool       { return v.compliant }
func (v storedVerdict) GetStatus() string        { return v.status }
func (v storedVerdict) GetReason() string        { return v.reason }
func (v storedVerdict) GetUpdatedAt() time.Time  { return v.updatedAt }

const (
	verdictTestCID = "baeareitestverdict00000000000000000000000000000000000000000001"
	trustedPolicy  = "cel:trusted"
	inForceVersion = "v1"
)

// fakeVerdicts is a VerdictSource holding whatever a test gives it.
type fakeVerdicts struct {
	rows []types.PolicyEvaluationObject
	err  error
	cids []string
}

func (f *fakeVerdicts) GetPolicyEvaluations(recordCID string) ([]types.PolicyEvaluationObject, error) {
	f.cids = append(f.cids, recordCID)

	return f.rows, f.err
}

func compliantRow() storedVerdict {
	return storedVerdict{
		policyID:  trustedPolicy,
		version:   inForceVersion,
		status:    types.PolicyEvalStatusEvaluated,
		compliant: true,
		updatedAt: time.Now(),
	}
}

func enforcing(ids ...string) types.PolicyEnforcement {
	enforcement := types.PolicyEnforcement{}
	for _, id := range ids {
		enforcement.Policies = append(enforcement.Policies, types.EnforcedPolicy{ID: id, Version: inForceVersion})
	}

	return enforcement
}

func verdictFor(t *testing.T, row storedVerdict, enforcement types.PolicyEnforcement) *policyv1.Verdict {
	t.Helper()

	verdicts := verdictsFor([]string{trustedPolicy},
		[]types.PolicyEvaluationObject{row}, versionsInForce(enforcement))
	require.Contains(t, verdicts, trustedPolicy)

	return verdicts[trustedPolicy]
}

// Every way of being stale has to reach the same false, because a policy
// reading compliant_and_current is asking one question and must not have to
// enumerate the ways the answer can be no.
func TestVerdictsFor_OnlyACurrentCompliantVerdictCounts(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		row    func(storedVerdict) storedVerdict
		want   policyv1.VerdictStatus
		counts bool
	}{
		"compliant at the version in force": {
			row:    func(r storedVerdict) storedVerdict { return r },
			want:   policyv1.VerdictStatus_VERDICT_STATUS_EVALUATED,
			counts: true,
		},
		"evaluated and not compliant": {
			row: func(r storedVerdict) storedVerdict {
				r.compliant = false

				return r
			},
			want: policyv1.VerdictStatus_VERDICT_STATUS_EVALUATED,
		},
		"compliant under a version since edited": {
			row: func(r storedVerdict) storedVerdict {
				r.version = "v0"

				return r
			},
			want: policyv1.VerdictStatus_VERDICT_STATUS_SUPERSEDED,
		},
		"superseded because the metadata moved": {
			row: func(r storedVerdict) storedVerdict {
				r.status = types.PolicyEvalStatusSuperseded
				r.compliant = false

				return r
			},
			want: policyv1.VerdictStatus_VERDICT_STATUS_SUPERSEDED,
		},
		"the evaluator failed": {
			row: func(r storedVerdict) storedVerdict {
				r.status = types.PolicyEvalStatusFailed
				r.compliant = false

				return r
			},
			want: policyv1.VerdictStatus_VERDICT_STATUS_FAILED,
		},
		"a status this build does not know": {
			row: func(r storedVerdict) storedVerdict {
				r.status = "invented"

				return r
			},
			want: policyv1.VerdictStatus_VERDICT_STATUS_UNSPECIFIED,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			verdict := verdictFor(t, tc.row(compliantRow()), enforcing(trustedPolicy))
			assert.Equal(t, tc.want, verdict.GetStatus())
			assert.Equal(t, tc.counts, verdict.GetCompliantAndCurrent())
		})
	}
}

// A compliant row under a policy no evaluator has registered is a conclusion
// about a rule nothing is running. It cannot count.
func TestVerdictsFor_AnUnregisteredPolicyHasNoCurrentVersion(t *testing.T) {
	t.Parallel()

	verdict := verdictFor(t, compliantRow(), types.PolicyEnforcement{
		Pending: []types.EnforcedPolicy{{ID: trustedPolicy}},
	})
	assert.Equal(t, policyv1.VerdictStatus_VERDICT_STATUS_SUPERSEDED, verdict.GetStatus())
	assert.False(t, verdict.GetCompliantAndCurrent())
}

// A policy still backfilling has a real version, so its verdicts read
// normally. Whether they may be acted on is the latch's business, not the
// map's.
func TestVerdictsFor_APendingPolicyStillReportsItsVerdict(t *testing.T) {
	t.Parallel()

	verdict := verdictFor(t, compliantRow(), types.PolicyEnforcement{
		Pending: []types.EnforcedPolicy{{ID: trustedPolicy, Version: inForceVersion}},
	})
	assert.Equal(t, policyv1.VerdictStatus_VERDICT_STATUS_EVALUATED, verdict.GetStatus())
	assert.True(t, verdict.GetCompliantAndCurrent())
}

// "Configured but not looked at yet" has to be tellable from "no such
// policy", which is what the entry being present at all says.
func TestVerdictsFor_AConfiguredPolicyWithNoRowIsUnevaluated(t *testing.T) {
	t.Parallel()

	verdicts := verdictsFor([]string{trustedPolicy, "opa:licensed"},
		[]types.PolicyEvaluationObject{compliantRow()}, versionsInForce(enforcing(trustedPolicy)))

	require.Contains(t, verdicts, "opa:licensed")
	assert.Equal(t, policyv1.VerdictStatus_VERDICT_STATUS_UNEVALUATED, verdicts["opa:licensed"].GetStatus())
	assert.False(t, verdicts["opa:licensed"].GetCompliantAndCurrent())
}

// A verdict under a policy nobody asked for answers no question being decided
// now, and leaving it in would let an expression read a key that is not
// configured.
func TestVerdictsFor_DropsRowsForPoliciesNoLongerConfigured(t *testing.T) {
	t.Parallel()

	verdicts := verdictsFor(nil,
		[]types.PolicyEvaluationObject{compliantRow()}, versionsInForce(enforcing(trustedPolicy)))

	assert.Empty(t, verdicts)
}

func registryReading(t *testing.T, expr string, source VerdictSource, enforcement types.PolicyEnforcement) *Registry {
	t.Helper()

	reg, err := NewRegistry(context.Background(),
		policyconfig.Evaluations{{
			Provider:    policyconfig.ProviderCEL,
			Name:        "trusted",
			Expressions: []string{`record.name != ""`},
		}},
		policyconfig.Operations{{
			Provider:    policyconfig.ProviderCEL,
			Ops:         []string{policyconfig.OpPull},
			Name:        "serve-checked-records",
			Expressions: []string{expr},
		}},
		"",
		WithVerdicts(source, func() types.PolicyEnforcement { return enforcement }),
	)
	require.NoError(t, err)

	return reg
}

// The latch: a policy requiring a verdict from an evaluation policy that has
// not covered the catalogue would hide every record not yet evaluated, which
// is most of them. It reports instead until the backfill completes.
func TestRegistryMayEnforce_HoldsOffUntilWhatItReadsHasBackfilled(t *testing.T) {
	t.Parallel()

	reading := verdictRefs{keys: []string{trustedPolicy}}

	pending := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`, &fakeVerdicts{},
		types.PolicyEnforcement{Pending: []types.EnforcedPolicy{{ID: trustedPolicy, Version: inForceVersion}}})
	assert.False(t, pending.mayEnforce(reading), "still backfilling")

	covered := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`, &fakeVerdicts{},
		enforcing(trustedPolicy))
	assert.True(t, covered.mayEnforce(reading), "backfill complete")
}

// A policy reading no verdict has nothing to wait on, so the latch must not
// catch it.
func TestRegistryMayEnforce_LeavesAPolicyReadingNoVerdictAlone(t *testing.T) {
	t.Parallel()

	reg := registryReading(t, `record_cid != ""`, &fakeVerdicts{}, types.PolicyEnforcement{})
	assert.True(t, reg.mayEnforce(verdictRefs{}))
}

// A reference whose keys could not be read might mean any policy, so it waits
// for all of them.
func TestRegistryMayEnforce_AnUnreadableReferenceWaitsForEveryPolicy(t *testing.T) {
	t.Parallel()

	pending := registryReading(t, `record_cid != ""`, &fakeVerdicts{},
		types.PolicyEnforcement{Pending: []types.EnforcedPolicy{{ID: trustedPolicy, Version: inForceVersion}}})
	assert.False(t, pending.mayEnforce(verdictRefs{all: true}))

	covered := registryReading(t, `record_cid != ""`, &fakeVerdicts{}, enforcing(trustedPolicy))
	assert.True(t, covered.mayEnforce(verdictRefs{all: true}))
}

// Reading verdicts costs a query, so an op whose policies read none must not
// pay it.
func TestRegistryDecide_ReadsVerdictsOnlyWhenAPolicyAsks(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	source := &fakeVerdicts{}
	reg := registryReading(t, `record_cid != ""`, source, enforcing(trustedPolicy))

	_, _, err := reg.Decide(context.Background(), policyconfig.OpPull, record)
	require.NoError(t, err)
	assert.Empty(t, source.cids, "no policy reads a verdict")

	asking := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`, source, enforcing(trustedPolicy))
	_, _, err = asking.Decide(context.Background(), policyconfig.OpPull, record)
	require.NoError(t, err)
	assert.Equal(t, []string{record.GetCid()}, source.cids)
}

// The gate as a policy: a record whose verdict does not count is refused.
func TestRegistryDecide_RefusesARecordWhoseVerdictDoesNotCount(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	expr := `verdicts["cel:trusted"].compliant_and_current`

	// No row at all: the evaluation has not reached this record yet.
	reg := registryReading(t, expr, &fakeVerdicts{}, enforcing(trustedPolicy))
	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPull, record)
	require.NoError(t, err)
	assert.False(t, ok, "a record with no verdict is not served")
	assert.NotEmpty(t, msgs)

	// A compliant verdict at the version in force.
	served := registryReading(t, expr,
		&fakeVerdicts{rows: []types.PolicyEvaluationObject{compliantRow()}}, enforcing(trustedPolicy))
	ok, _, err = served.Decide(context.Background(), policyconfig.OpPull, record)
	require.NoError(t, err)
	assert.True(t, ok, "a record with a current compliant verdict is served")
}

// The latch, through a real decision. The same policy that refuses above has
// to let the record through while the evaluation policy it reads is still
// covering the catalogue, or switching a policy on empties the catalogue.
func TestRegistryDecide_ReportsRatherThanRefusesWhileBackfilling(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	reg := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`, &fakeVerdicts{},
		types.PolicyEnforcement{Pending: []types.EnforcedPolicy{{ID: trustedPolicy, Version: inForceVersion}}})

	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPull, record)
	require.NoError(t, err)
	assert.True(t, ok, "held to reporting until what it reads has backfilled")
	assert.Empty(t, msgs)
}

// A Rego policy reads the same map, including the status by name rather than
// by number.
func TestRegistryDecide_RegoReadsTheSameVerdicts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gate.rego"), []byte(`package gate

allow if {
	input.verdicts["cel:trusted"].status == "VERDICT_STATUS_EVALUATED"
	input.verdicts["cel:trusted"].compliant_and_current
}
`), 0o600))

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	decide := func(rows []types.PolicyEvaluationObject) bool {
		reg, err := NewRegistry(context.Background(),
			policyconfig.Evaluations{{
				Provider:    policyconfig.ProviderCEL,
				Name:        "trusted",
				Expressions: []string{`record.name != ""`},
			}},
			policyconfig.Operations{{
				Provider: policyconfig.ProviderOPA,
				Ops:      []string{policyconfig.OpPull},
				Name:     "gate",
				File:     "gate.rego",
			}},
			dir,
			WithVerdicts(&fakeVerdicts{rows: rows}, func() types.PolicyEnforcement { return enforcing(trustedPolicy) }),
		)
		require.NoError(t, err)

		ok, _, err := reg.Decide(context.Background(), policyconfig.OpPull, record)
		require.NoError(t, err)

		return ok
	}

	assert.False(t, decide(nil), "no verdict")
	assert.True(t, decide([]types.PolicyEvaluationObject{compliantRow()}), "a current compliant verdict")
}
