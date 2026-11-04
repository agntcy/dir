// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/config"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parityPolicy is the evaluation policy both gates decide on, named so that
// the registry's ID for it matches what the database records verdicts under,
// and parityVersion is the version in force.
const (
	parityPolicyName = "a"
	parityPolicy     = "cel:" + parityPolicyName
	parityVersion    = "v1"
)

// parityExpression is the pull operation policy that reads the same verdict
// the SQL gate selects on. It is written here rather than in a file so that
// what it reads sits next to what it is compared against.
const parityExpression = `verdicts["` + parityPolicy + `"].compliant_and_current`

// unindexedCID is a record this node does not hold.
const unindexedCID = "baeareigatenone000000000000000000000000000000000000000000000000"

// parityNode is one indexed record with whatever verdict the case calls for,
// read through both gates.
type parityNode struct {
	db  *gormdb.DB
	cid string
}

func newParityNode(t *testing.T) parityNode {
	t.Helper()

	db := newTestDatabase(t)

	record := corev1.New(&typesv1alpha1.Record{Name: "parity", Version: "1.0.0", SchemaVersion: "0.7.0"})

	adapter, err := record.Decode()
	require.NoError(t, err)
	require.NoError(t, db.AddRecord(adapter))
	require.NoError(t, db.RegisterPolicyVersion(parityPolicy, parityVersion, nil))

	return parityNode{db: db, cid: record.GetCid()}
}

// verdict records what the evaluation policy concluded, at version.
func (n parityNode) verdict(t *testing.T, evalStatus string, compliant bool, version string) {
	t.Helper()

	require.NoError(t, n.db.UpsertPolicyEvaluation(&gormdb.PolicyEvaluation{
		RecordCID:     n.cid,
		PolicyID:      parityPolicy,
		PolicyVersion: version,
		Status:        evalStatus,
		Compliant:     compliant,
	}))
}

// served reports whether the pull gate let cid through, insisting that a
// refusal is the one a caller is meant to see and not some other failure.
func (n parityNode) served(t *testing.T, mode policyconfig.Mode, cid string) bool {
	t.Helper()

	cfg := pullOnlyConfig()
	cfg.Policy.Operations[0].Mode = mode

	err := pullOnlyGate(t, cfg, n, cid)
	if err == nil {
		return true
	}

	require.Equal(t, codes.PermissionDenied, status.Code(err))

	return false
}

// The gate's answer for every state a verdict can be in.
//
// These cases were written to pin the replaced SQL gate and the pull policy
// to the same answers; that gate is gone, and what is left is the behaviour
// it was held to.
//
// Most states serve, and not because the gate is lax. A record is only
// refused once the policy is enforced, and a policy is only enforced once its
// verdicts cover the catalogue. A verdict that failed, was superseded, or was
// reached under a since-edited version does not cover the record it is for,
// so the policy is still backfilling and refuses nothing. The one state that
// refuses an indexed record is the one that says so: an evaluated verdict
// that found it non-compliant.
func TestPullGate_AnswersForEveryVerdictState(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		seed   func(t *testing.T, n parityNode)
		served bool
	}{
		{
			name: "a compliant verdict at the version in force",
			seed: func(t *testing.T, n parityNode) {
				t.Helper()
				n.verdict(t, types.PolicyEvalStatusEvaluated, true, parityVersion)
			},
			served: true,
		},
		{
			name: "a verdict that found the record non-compliant",
			seed: func(t *testing.T, n parityNode) {
				t.Helper()
				n.verdict(t, types.PolicyEvalStatusEvaluated, false, parityVersion)
			},
			served: false,
		},
		{
			name: "a compliant verdict under a version since edited",
			seed: func(t *testing.T, n parityNode) {
				t.Helper()
				n.verdict(t, types.PolicyEvalStatusEvaluated, true, "v0")
			},
			served: true,
		},
		{
			name: "a verdict the evaluator could not reach",
			seed: func(t *testing.T, n parityNode) {
				t.Helper()
				n.verdict(t, types.PolicyEvalStatusFailed, false, parityVersion)
			},
			served: true,
		},
		{
			name: "a verdict superseded by a change in what it read",
			seed: func(t *testing.T, n parityNode) {
				t.Helper()
				n.verdict(t, types.PolicyEvalStatusSuperseded, true, parityVersion)
			},
			served: true,
		},
		{
			name:   "no verdict yet",
			seed:   func(t *testing.T, n parityNode) { t.Helper() },
			served: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, mode := range []policyconfig.Mode{policyconfig.ModeEnforce, policyconfig.ModeShadow} {
				t.Run(string(mode), func(t *testing.T) {
					t.Parallel()

					// Shadow reports what it would refuse and serves it anyway.
					want := tc.served || mode == policyconfig.ModeShadow

					n := newParityNode(t)
					tc.seed(t, n)
					assert.Equal(t, want, n.served(t, mode, n.cid))
				})
			}
		})
	}
}

// A record this node does not hold has no verdict and never will, so once the
// catalogue is covered and the policy is enforced, the gate refuses it. A
// refusal needs a policy that is actually enforced, which is the whole of
// what separates the two modes here.
func TestPullGate_RefusesAnUnindexedRecord(t *testing.T) {
	t.Parallel()

	n := newParityNode(t)
	n.verdict(t, types.PolicyEvalStatusEvaluated, true, parityVersion)

	assert.False(t, n.served(t, policyconfig.ModeEnforce, unindexedCID))
	assert.True(t, n.served(t, policyconfig.ModeShadow, unindexedCID), "shadow serves it")
}

// pullOnlyConfig configures the pull policy and the evaluation it reads, and
// nothing else.
func pullOnlyConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Authz.Enabled = true
	cfg.Policy.Evaluations = policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        parityPolicyName,
		Expressions: []string{`record.name != ""`},
	}}
	cfg.Policy.Operations = policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPull},
		Name:        "serve-checked-records",
		Expressions: []string{parityExpression},
	}}

	return cfg
}

// pullOnlyGate wires cfg afresh and asks the pull gate about cid, so that
// each step below sees enforcement resolved against the database as it stands
// at that moment.
func pullOnlyGate(t *testing.T, cfg *config.Config, n parityNode, cid string) error {
	t.Helper()

	wired, err := wirePolicies(t.Context(), cfg, n.db, nil)
	require.NoError(t, err)

	//nolint:wrapcheck // the gate's own error is what the caller asserts on.
	return wired.Gate.Check(t.Context(), cid)
}

// The pull policy latches, reads its verdicts, and enforces, with no
// enforcement block present at all: the arrangement left once it is deleted.
//
// Everything above configures the list, because until now the list fed both
// the latch and the versions a verdict is judged current against. Without it
// the latch never released, so a policy reading that verdict stayed in shadow
// for ever; and no version was in force, so every verdict read as superseded.
// The two faults cancel, the first hiding the second, which is why all three
// steps are here. A gate that refuses from the start is not latching, and one
// that refuses a compliant record is not reading its verdicts.
func TestPullGate_LatchesAndEnforcesWithoutTheEnforcementBlock(t *testing.T) {
	t.Parallel()

	n := newParityNode(t)
	cfg := pullOnlyConfig()

	require.NoError(t, pullOnlyGate(t, cfg, n, n.cid),
		"no verdict yet, so the catalogue is uncovered and the policy only reports")

	// A verdict covers the catalogue, so the policy enforces. It serves,
	// which it can only do if the version the verdict was reached under is
	// the version in force.
	n.verdict(t, types.PolicyEvalStatusEvaluated, true, parityVersion)
	require.NoError(t, pullOnlyGate(t, cfg, n, n.cid),
		"a compliant verdict at the version in force counts")

	// The same enforcing policy, against a record whose verdict says no.
	n.verdict(t, types.PolicyEvalStatusEvaluated, false, parityVersion)

	err := pullOnlyGate(t, cfg, n, n.cid)
	require.Error(t, err, "the verdict says non-compliant and the policy is enforcing")
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}
