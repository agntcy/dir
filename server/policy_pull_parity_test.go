// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	gormdb "github.com/agntcy/dir/server/database/gorm"
	"github.com/agntcy/dir/server/policy"
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

// sqlGate is the gate as it is today: the enforcement config, applied to the
// served view of the database by the query builder.
func (n parityNode) sqlGate(t *testing.T, mode policyconfig.Mode, cid string) bool {
	t.Helper()

	served, _, err := servedDatabase(n.db,
		policyconfig.EnforcementConfig{Fetch: mode, Policies: []string{parityPolicy}}, true, nil)
	require.NoError(t, err)

	ok, err := served.IsRecordServable(cid)
	require.NoError(t, err)

	return ok
}

// policyGate is the gate as an operation policy at pull: the same verdict,
// read by a configured expression rather than by the query builder.
//
// Enforcement is resolved as the apiserver resolves it, so that the backfill
// latch applies to this gate exactly as it applies to the other one.
func (n parityNode) policyGate(t *testing.T, mode policyconfig.Mode, cid string) bool {
	t.Helper()

	_, enforced, err := servedDatabase(n.db,
		policyconfig.EnforcementConfig{Fetch: policyconfig.ModeEnforce, Policies: []string{parityPolicy}}, true, nil)
	require.NoError(t, err)

	registry, err := policy.NewRegistry(t.Context(),
		policyconfig.Evaluations{{
			Provider:    policyconfig.ProviderCEL,
			Name:        parityPolicyName,
			Expressions: []string{`record.name != ""`},
		}},
		policyconfig.Operations{{
			Provider:    policyconfig.ProviderCEL,
			Ops:         []string{policyconfig.OpPull},
			Name:        "serve-checked-records",
			Mode:        mode,
			Expressions: []string{parityExpression},
		}},
		"", policy.WithVerdicts(n.db, enforced))
	require.NoError(t, err)

	err = policy.NewRecordGate(nil, registry).Check(t.Context(), cid)
	if err == nil {
		return true
	}

	require.Equal(t, codes.PermissionDenied, status.Code(err))

	return false
}

// The two gates answer alike for every state a verdict can be in. This is what
// makes replacing one with the other a change of mechanism rather than of
// behaviour: whichever decides, the same record is served or refused.
//
// Most states serve, and not because the gate is lax. A record is only
// refused once the policy is enforced, and a policy is only enforced once its
// verdicts cover the catalogue. A verdict that failed, was superseded, or was
// reached under a since-edited version does not cover the record it is for,
// so the policy is still backfilling and refuses nothing. The one state that
// refuses an indexed record is the one that says so: an evaluated verdict
// that found it non-compliant.
func TestPullGateParity_BothGatesAgree(t *testing.T) {
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
					assert.Equal(t, want, n.sqlGate(t, mode, n.cid), "the gate as it is today")

					n = newParityNode(t)
					tc.seed(t, n)
					assert.Equal(t, want, n.policyGate(t, mode, n.cid), "the gate as an operation policy")
				})
			}
		})
	}
}

// A record this node does not hold has no verdict and never will, so once the
// catalogue is covered and the policy is enforced, both gates refuse it. This
// is the other half of the parity: a refusal needs a policy that is actually
// enforced, and nothing else about the two gates differs.
func TestPullGateParity_BothGatesRefuseAnUnindexedRecord(t *testing.T) {
	t.Parallel()

	n := newParityNode(t)
	n.verdict(t, types.PolicyEvalStatusEvaluated, true, parityVersion)

	assert.False(t, n.sqlGate(t, policyconfig.ModeEnforce, unindexedCID), "the gate as it is today")
	assert.False(t, n.policyGate(t, policyconfig.ModeEnforce, unindexedCID), "the gate as an operation policy")

	assert.True(t, n.sqlGate(t, policyconfig.ModeShadow, unindexedCID), "shadow serves it")
	assert.True(t, n.policyGate(t, policyconfig.ModeShadow, unindexedCID), "shadow serves it")
}
