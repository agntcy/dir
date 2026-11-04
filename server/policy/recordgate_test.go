// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"errors"
	"testing"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const gatedCID = "baeareigatetest00000000000000000000000000000000000000000000000"

// excluded records what the gate reports, in the order it reports it.
type excluded struct {
	modes []policyconfig.Mode
}

func (e *excluded) PullExcluded(mode policyconfig.Mode) { e.modes = append(e.modes, mode) }

// A caller refused by a policy is told the record is not available under this
// node's content policy, with no policy name and no reason, and a node
// configuring nothing at pull serves whatever it holds.
func TestRecordGateCheck_APolicyCanRefuse(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		gate *RecordGate
		code codes.Code
	}{
		{"the policy allows", NewRecordGate(registryReading(t, `record_cid != ""`,
			&fakeVerdicts{}, enforcing(trustedPolicy)), nil), codes.OK},
		{"the policy refuses", NewRecordGate(registryReading(t, `verdicts["cel:trusted"].compliant_and_current`,
			&fakeVerdicts{}, enforcing(trustedPolicy)), nil), codes.PermissionDenied},
		{"nothing is configured", NewRecordGate(nil, nil), codes.OK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.gate.Check(t.Context(), gatedCID)

			assert.Equal(t, tc.code, status.Code(err))
		})
	}
}

// A check that cannot be answered fails closed, and is reported as a failed
// check rather than as an excluded record: the two mean different things to
// whoever is reading the logs.
//
// Unless nothing would have acted on it. A shadow policy exists to be
// watched, and a policy being watched must not be able to fail a read, so
// the record is served and the failure only logged.
func TestRecordGateCheck_AnUnanswerableCheckFailsClosedOnlyWhenItCounts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mode policyconfig.Mode
		code codes.Code
	}{
		{policyconfig.ModeEnforce, codes.Internal},
		{policyconfig.ModeShadow, codes.OK},
	} {
		t.Run(string(tc.mode), func(t *testing.T) {
			t.Parallel()

			counted := &excluded{}
			registry := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`,
				&fakeVerdicts{err: errors.New("database unavailable")}, enforcing(trustedPolicy))
			registry.byOp[policyconfig.OpPull][0].mode = tc.mode

			err := NewRecordGate(registry, counted).Check(t.Context(), gatedCID)

			assert.Equal(t, tc.code, status.Code(err))
			assert.Empty(t, counted.modes, "a check that did not happen excluded nothing")
		})
	}
}

// The latch decides this too: a policy configured to enforce but still
// backfilling acts on nothing, so a check it cannot make fails no read.
func TestRecordGateCheck_AnUnanswerableCheckUnderTheLatchIsTolerated(t *testing.T) {
	t.Parallel()

	gate := NewRecordGate(registryReading(t, `verdicts["cel:trusted"].compliant_and_current`,
		&fakeVerdicts{err: errors.New("database unavailable")}, backfilling(trustedPolicy)), nil)

	assert.NoError(t, gate.Check(t.Context(), gatedCID))
}

// One refused read is one increment, counted under the mode that refused.
//
// A shadow policy reaches the counter with the record still served, which is
// the only way the would-have-excluded series is ever anything but zero: a
// shadow refusal leaves no other trace but its log line.
func TestRecordGateCheck_ARefusalIsCountedUnderItsMode(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		mode policyconfig.Mode
		code codes.Code
		want []policyconfig.Mode
	}{
		{policyconfig.ModeEnforce, codes.PermissionDenied, []policyconfig.Mode{policyconfig.ModeEnforce}},
		{policyconfig.ModeShadow, codes.OK, []policyconfig.Mode{policyconfig.ModeShadow}},
	} {
		t.Run(string(tc.mode), func(t *testing.T) {
			t.Parallel()

			counted := &excluded{}
			registry := registryReading(t, `verdicts["cel:trusted"].compliant_and_current`,
				&fakeVerdicts{}, enforcing(trustedPolicy))
			registry.byOp[policyconfig.OpPull][0].mode = tc.mode

			err := NewRecordGate(registry, counted).Check(t.Context(), gatedCID)

			assert.Equal(t, tc.code, status.Code(err))
			assert.Equal(t, tc.want, counted.modes)
		})
	}
}

// Nothing is counted for a record the policies admit.
func TestRecordGateCheck_AServedRecordIsNotCounted(t *testing.T) {
	t.Parallel()

	counted := &excluded{}
	gate := NewRecordGate(registryReading(t, `record_cid != ""`, &fakeVerdicts{}, enforcing(trustedPolicy)), counted)

	require.NoError(t, gate.Check(t.Context(), gatedCID))
	assert.Empty(t, counted.modes)
}

// twoPullPolicies builds a registry with two policies at pull: the first
// reading no verdict, the second reading one. Each carries the given mode,
// so a test can say which of the two would act.
func twoPullPolicies(t *testing.T, source VerdictSource, modes [2]policyconfig.Mode) *Registry {
	t.Helper()

	reg, err := NewRegistry(context.Background(),
		policyconfig.Evaluations{{
			Provider:    policyconfig.ProviderCEL,
			Name:        "trusted",
			Expressions: []string{`record.name != ""`},
		}},
		policyconfig.Operations{
			{
				Provider:    policyconfig.ProviderCEL,
				Ops:         []string{policyconfig.OpPull},
				Name:        "reads-no-verdict",
				Mode:        modes[0],
				Expressions: []string{`record_cid != ""`},
			},
			{
				Provider:    policyconfig.ProviderCEL,
				Ops:         []string{policyconfig.OpPull},
				Name:        "reads-a-verdict",
				Mode:        modes[1],
				Expressions: []string{`verdicts["cel:trusted"].compliant_and_current`},
			},
		},
		"",
		WithVerdicts(source, func() types.PolicyEnforcement { return enforcing(trustedPolicy) }),
	)
	require.NoError(t, err)

	return reg
}

// Whether an unreadable verdict store fails a read is not "is anything
// enforcing" but "is anything that reads verdicts enforcing". An enforcing
// policy that reads none decides the same either way, so it must not be the
// reason a read fails closed over a question it never asked.
func TestRecordGateCheck_AnEnforcingPolicyThatReadsNoVerdictDoesNotFailTheRead(t *testing.T) {
	t.Parallel()

	registry := twoPullPolicies(t, &fakeVerdicts{err: errors.New("database unavailable")},
		[2]policyconfig.Mode{policyconfig.ModeEnforce, policyconfig.ModeShadow})

	err := NewRecordGate(registry, nil).Check(t.Context(), gatedCID)

	assert.Equal(t, codes.OK, status.Code(err),
		"only the shadow policy reads verdicts, so nothing that acts was deprived of one")
}

// The counterpart: once the policy that reads verdicts is the enforcing one,
// the same failure does fail the read.
func TestRecordGateCheck_AnEnforcingPolicyThatReadsVerdictsFailsTheRead(t *testing.T) {
	t.Parallel()

	registry := twoPullPolicies(t, &fakeVerdicts{err: errors.New("database unavailable")},
		[2]policyconfig.Mode{policyconfig.ModeShadow, policyconfig.ModeEnforce})

	err := NewRecordGate(registry, nil).Check(t.Context(), gatedCID)

	assert.Equal(t, codes.Internal, status.Code(err))
}
