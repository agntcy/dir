// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry_Empty(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, nil, "")
	require.NoError(t, err)
	require.Empty(t, reg.OperationIDs(policyconfig.OpPush))
	require.Empty(t, reg.OperationIDs(policyconfig.OpIndex))
}

func TestNewRegistry_RejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider: "unknown",
		Ops:      []string{policyconfig.OpPush},
		Name:     "x",
	}}, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), `unsupported provider "unknown"`)
}

func TestNewRegistry_RejectsEmptySchemaURL(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider: policyconfig.ProviderOASF,
		Ops:      []string{policyconfig.OpPush},
		Name:     "schema",
	}}, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "schema_url is required")
}

func TestNewRegistry_MultiplePerOp(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{
		{
			Provider:  policyconfig.ProviderOASF,
			Ops:       []string{policyconfig.OpPush},
			Name:      "schema-a",
			SchemaURL: "https://schema.example.com/a",
		},
		{
			Provider:  policyconfig.ProviderOASF,
			Ops:       []string{policyconfig.OpPush, policyconfig.OpIndex},
			Name:      "schema-b",
			SchemaURL: "https://schema.example.com/b",
		},
	}, "")
	require.NoError(t, err)
	require.Len(t, reg.OperationIDs(policyconfig.OpPush), 2)
	require.Len(t, reg.OperationIDs(policyconfig.OpIndex), 1)
	require.Empty(t, reg.OperationIDs(policyconfig.OpAutosync))
}

func TestNewRegistry_SingleEntryMultipleOps(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:  policyconfig.ProviderOASF,
		Ops:       []string{policyconfig.OpPush, policyconfig.OpIndex},
		Name:      "schema",
		SchemaURL: "https://schema.oasf.outshift.com",
	}}, "")
	require.NoError(t, err)
	require.Len(t, reg.OperationIDs(policyconfig.OpPush), 1)
	require.Len(t, reg.OperationIDs(policyconfig.OpIndex), 1)
	require.Empty(t, reg.OperationIDs(policyconfig.OpAutosync))
}

type fakeValidator struct {
	valid bool
	msgs  []string
}

func (f fakeValidator) ValidateOperation(context.Context, *policyv1.OperationInput) (bool, []string, []string, error) {
	return f.valid, f.msgs, nil, nil
}

func (f fakeValidator) ValidateEvaluation(context.Context, *policyv1.RecordState) (bool, []string, []string, error) {
	return false, nil, nil, nil
}

func TestRun_EmptyIsValid(t *testing.T) {
	t.Parallel()

	var reg *Registry

	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
}

// countingValidator rejects every record and counts how often it was asked,
// so a mode that must not run can be told from one that runs and is ignored.
type countingValidator struct {
	calls int
}

func (c *countingValidator) ValidateOperation(context.Context, *policyv1.OperationInput) (bool, []string, []string, error) {
	c.calls++

	return false, []string{"rejected"}, nil, nil
}

func (c *countingValidator) ValidateEvaluation(context.Context, *policyv1.RecordState) (bool, []string, []string, error) {
	return false, nil, nil, nil
}

// A policy's mode decides whether its rejection reaches the caller. An unset
// mode enforces, because an operation policy was always acted on and no
// shipped configuration sets one.
func TestDecide_AppliesEachPolicyMode(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "test"})

	tests := []struct {
		mode      policyconfig.Mode
		wantValid bool
		wantRun   bool
	}{
		{policyconfig.ModeOff, true, false},
		{policyconfig.ModeShadow, true, true},
		{policyconfig.ModeEnforce, false, true},
		{"", false, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.mode), func(t *testing.T) {
			t.Parallel()

			v := &countingValidator{}

			mode := tc.mode
			if mode == "" {
				mode = policyconfig.ModeEnforce
			}

			reg := &Registry{byOp: map[string][]operationPolicy{
				policyconfig.OpPush: {{id: "cel:p", mode: mode, validator: v}},
			}}

			ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush, record)
			require.NoError(t, err)
			require.Equal(t, tc.wantValid, ok, msgs)

			if tc.wantRun {
				require.Equal(t, 1, v.calls, "the policy should have been asked")
			} else {
				require.Zero(t, v.calls, "an off policy should not be run at all")
			}
		})
	}
}

// An unset mode has to reach the registry as enforce, because the OASF schema
// validator in every shipped configuration omits it.
func TestNewRegistry_UnsetModeEnforces(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush},
		Name:        "never",
		Expressions: []string{`false`},
	}}, "")
	require.NoError(t, err)
	require.Equal(t, []string{"cel:never"}, reg.OperationIDs(policyconfig.OpPush))

	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush,
		corev1.New(&typesv1alpha1.Record{Name: "test"}))
	require.NoError(t, err)
	require.False(t, ok)
	require.NotEmpty(t, msgs)
}

func TestRun_CollectsFailures(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "test"})
	reg := With(policyconfig.OpPush,
		fakeValidator{valid: true},
		fakeValidator{valid: false, msgs: []string{"first"}},
		fakeValidator{valid: false, msgs: []string{"second"}},
	)

	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush, record)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, []string{"ERROR: first", "ERROR: second"}, msgs)
}

// A policy reading verdicts at an operation that has none would read an empty
// map and refuse everything while appearing to decide. It is refused at load
// time instead, by name, so the operator is told which op is the problem.
func TestNewRegistry_RejectsVerdictsWhereThereAreNone(t *testing.T) {
	t.Parallel()

	for _, op := range []string{policyconfig.OpPush, policyconfig.OpAutosync, policyconfig.OpIndex} {
		_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
			Provider:    policyconfig.ProviderCEL,
			Ops:         []string{op},
			Name:        "serve-checked-records",
			Expressions: []string{`verdicts["cel:trusted"].compliant_and_current`},
		}}, "")
		require.ErrorContains(t, err, `reads verdicts, which only op "pull" has`, op)
		require.ErrorContains(t, err, op)
	}
}

// Listing pull alongside an operation that has no verdicts is the same
// mistake, and is rejected for the operation that cannot answer.
func TestNewRegistry_RejectsVerdictsAtPullPlusAnotherOp(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPull, policyconfig.OpPush},
		Name:        "serve-checked-records",
		Expressions: []string{`verdicts["cel:trusted"].compliant_and_current`},
	}}, "")
	require.ErrorContains(t, err, `op "push" decides before any verdict exists`)
}

func TestNewRegistry_AcceptsVerdictsAtPull(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPull},
		Name:        "serve-checked-records",
		Expressions: []string{`verdicts["cel:trusted"].compliant_and_current`},
	}}, "", WithVerdicts(&fakeVerdicts{}, func() types.PolicyEnforcement { return types.PolicyEnforcement{} }))
	require.NoError(t, err)
	require.Equal(t, []string{"cel:serve-checked-records"}, reg.OperationIDs(policyconfig.OpPull))
}

// A pull decision is reached before the record is read, so a policy that
// reads the record's content there would decide on an absence. It is refused
// at load time, and told where such a rule belongs.
func TestNewRegistry_RejectsReadingTheRecordAtPull(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		entry policyconfig.Operation
	}{
		{
			name: "cel",
			entry: policyconfig.Operation{
				Provider:    policyconfig.ProviderCEL,
				Ops:         []string{policyconfig.OpPull},
				Name:        "no-drafts",
				Expressions: []string{`record.name != ""`},
			},
		},
		{
			name: "oasf",
			entry: policyconfig.Operation{
				Provider:  policyconfig.ProviderOASF,
				Ops:       []string{policyconfig.OpPull},
				Name:      "schema",
				SchemaURL: "https://schema.oasf.outshift.com",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{tc.entry}, "")

			require.ErrorContains(t, err, `reads the record at op "pull"`)
			require.ErrorContains(t, err, "read its verdict here")
		})
	}
}

// Reading the record anywhere else is what an operation policy is for, and is
// left alone.
func TestNewRegistry_LeavesTheRecordReadableEverywhereElse(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush, policyconfig.OpIndex},
		Name:        "no-drafts",
		Expressions: []string{`record.name != ""`},
	}}, "")
	require.NoError(t, err)
}

// A policy that reads no verdict is unaffected, whatever op it runs at: the
// check is about reading verdicts, not about which ops exist.
func TestNewRegistry_LeavesPoliciesThatReadNoVerdictAlone(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush, policyconfig.OpIndex},
		Name:        "named-records",
		Expressions: []string{`record.name != ""`},
	}}, "")
	require.NoError(t, err)
}
