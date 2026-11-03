// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/agntcy/dir/server/policy/predicate"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// searchTestEvaluations are the evaluation policies the expressions below
// name. A verdict key is resolved against the configured policies as it
// compiles, so a key no test configures is refused before the subset is
// asked about it.
var searchTestEvaluations = []string{"cel:a", "cel:b", "cel:c", "cel:trusted", "id"}

// compileSearch compiles expressions as an operation policy and returns what
// a search would enforce, or why it would not.
func compileSearch(t *testing.T, exprs ...string) (predicate.Node, error) {
	t.Helper()

	v, err := newCELValidatorWithIDs(exprs, layerOperation, searchTestEvaluations)
	require.NoError(t, err)

	return v.predicate, v.predicateErr
}

func searchTerm(id string) predicate.Node {
	return predicate.Verdict{PolicyID: id}
}

func TestCELSearchPredicate_WhatTheSubsetAccepts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		exprs []string
		want  predicate.Node
	}{
		{
			name:  "one verdict",
			exprs: []string{`verdicts["cel:trusted"].compliant_and_current`},
			want:  searchTerm("cel:trusted"),
		},
		{
			name:  "a conjunction",
			exprs: []string{`verdicts["cel:a"].compliant_and_current && verdicts["cel:b"].compliant_and_current`},
			want:  predicate.All{Parts: []predicate.Node{searchTerm("cel:a"), searchTerm("cel:b")}},
		},
		{
			name:  "a disjunction",
			exprs: []string{`verdicts["cel:a"].compliant_and_current || verdicts["cel:b"].compliant_and_current`},
			want:  predicate.Any{Parts: []predicate.Node{searchTerm("cel:a"), searchTerm("cel:b")}},
		},
		{
			name: "grouping is kept",
			exprs: []string{`verdicts["cel:a"].compliant_and_current && ` +
				`(verdicts["cel:b"].compliant_and_current || verdicts["cel:c"].compliant_and_current)`},
			want: predicate.All{Parts: []predicate.Node{
				searchTerm("cel:a"),
				predicate.Any{Parts: []predicate.Node{searchTerm("cel:b"), searchTerm("cel:c")}},
			}},
		},
		{
			// A policy's expressions all have to hold, which is the same
			// reading every other op gives them.
			name: "several expressions are conjoined",
			exprs: []string{
				`verdicts["cel:a"].compliant_and_current`,
				`verdicts["cel:b"].compliant_and_current`,
			},
			want: predicate.All{Parts: []predicate.Node{searchTerm("cel:a"), searchTerm("cel:b")}},
		},
		{
			// The same policy named twice is not worth detecting: it renders
			// two identical conditions, which no database minds.
			name:  "the same verdict twice",
			exprs: []string{`verdicts["cel:a"].compliant_and_current && verdicts["cel:a"].compliant_and_current`},
			want:  predicate.All{Parts: []predicate.Node{searchTerm("cel:a"), searchTerm("cel:a")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := compileSearch(t, tt.exprs...)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Everything outside the subset is refused, and the refusal names what it
// refused so the author can see which part of the expression to change.
//
// None of these is refused by a check that looks for it. They are refused
// because the walk has no case that handles them, which is what makes
// widening the subset a deliberate act rather than an oversight.
func TestCELSearchPredicate_WhatTheSubsetRefuses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		expr    string
		wantErr string
	}{
		{
			// Would compile to no condition at all: a policy that is loaded,
			// reported as enforcing, and filtering nothing.
			name:    "a constant",
			expr:    `true`,
			wantErr: "expected a verdict",
		},
		{
			name:    "negation",
			expr:    `!verdicts["cel:a"].compliant_and_current`,
			wantErr: `"!_" is not available at search`,
		},
		{
			name:    "another field of the verdict",
			expr:    `verdicts["cel:a"].compliant`,
			wantErr: "can read only compliant_and_current of a verdict, not compliant",
		},
		{
			name:    "a comparison",
			expr:    `verdicts["cel:a"].policy_version == "v1"`,
			wantErr: `"_==_" is not available at search`,
		},
		{
			name:    "the record",
			expr:    `record.name != ""`,
			wantErr: `"_!=_" is not available at search`,
		},
		{
			name:    "the record CID",
			expr:    `record_cid != ""`,
			wantErr: `"_!=_" is not available at search`,
		},
		{
			name:    "the wall clock",
			expr:    `now < timestamp("2030-01-01T00:00:00Z")`,
			wantErr: `"_<_" is not available at search`,
		},
		{
			name:    "the operation context",
			expr:    `context.source_peer_id == "p"`,
			wantErr: `"_==_" is not available at search`,
		},
		{
			name:    "presence",
			expr:    `has(verdicts["cel:a"].compliant_and_current)`,
			wantErr: "has() is not available at search",
		},
		{
			// Refused whole: one part being expressible does not make the
			// policy expressible.
			name:    "one refused part of a conjunction",
			expr:    `verdicts["cel:a"].compliant_and_current && record_cid != ""`,
			wantErr: `"_!=_" is not available at search`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := compileSearch(t, tt.expr)
			require.ErrorContains(t, err, tt.wantErr)
			require.ErrorIs(t, err, errNotCompilable)
			assert.Nil(t, got, "a refused expression must not leave a predicate behind")
		})
	}
}

// A verdict key the compiler cannot read is refused before the subset is
// consulted, and at every op rather than only at search: an expression that
// hides which evaluation policies it depends on does not compile at all.
//
// The subset still refuses these on its own, which is why the refusals in
// celVerdictTerm stay. This records that nothing configurable reaches them,
// so a change to either side is not silently relying on the other.
func TestCELSearchPredicate_AnUnreadableKeyNeverReachesTheSubset(t *testing.T) {
	t.Parallel()

	for name, expr := range map[string]string{
		"a verdict key that is not a literal": `verdicts[record_cid].compliant_and_current`,
		"iteration":                           `["cel:a"].all(id, verdicts[id].compliant_and_current)`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := newCELValidatorWithIDs([]string{expr}, layerOperation, searchTestEvaluations)
			require.Error(t, err)
			assert.NotErrorIs(t, err, errNotCompilable, "refused before the subset is asked")
		})
	}
}

// Which expression was refused, because a policy's expressions are a list the
// author has to find their way back to.
func TestCELSearchPredicate_NamesTheExpressionItRefused(t *testing.T) {
	t.Parallel()

	_, err := compileSearch(t,
		`verdicts["cel:a"].compliant_and_current`,
		`verdicts["cel:b"].compliant_and_current`,
		`record_cid != ""`,
	)

	require.ErrorContains(t, err, "expressions[2]")
}

// A policy that cannot be compiled is refused when it is loaded, not when a
// search runs: a search has no way to report that its policy was unusable
// other than returning the wrong records.
func TestNewRegistry_RefusesASearchPolicyItCannotCompile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation policyconfig.Operation
		wantErr   string
	}{
		{
			name: "an expression outside the subset",
			operation: policyconfig.Operation{
				Provider:    policyconfig.ProviderCEL,
				Ops:         []string{policyconfig.OpSearch},
				Name:        "serve-checked-records",
				Expressions: []string{`record_cid != ""`},
			},
			wantErr: `cannot be used at op "search"`,
		},
		{
			// The record is refused with the advice that applies, rather than
			// as an unsupported construct.
			name: "the record",
			operation: policyconfig.Operation{
				Provider:    policyconfig.ProviderCEL,
				Ops:         []string{policyconfig.OpSearch},
				Name:        "serve-named-records",
				Expressions: []string{`record.name != ""`},
			},
			wantErr: `reads the record at op "search"`,
		},
		{
			// Listed at search alongside an op it would be fine at.
			name: "compilable at search, listed at both",
			operation: policyconfig.Operation{
				Provider:    policyconfig.ProviderCEL,
				Ops:         []string{policyconfig.OpPull, policyconfig.OpSearch},
				Name:        "serve-recent-records",
				Expressions: []string{`now < timestamp("2030-01-01T00:00:00Z")`},
			},
			wantErr: `cannot be used at op "search"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewRegistry(context.Background(), nil,
				policyconfig.Operations{tt.operation}, "", WithVerdicts(&fakeVerdicts{}, func() types.PolicyEnforcement { return types.PolicyEnforcement{} }))

			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// A policy at both ops is one policy: the same expressions are evaluated at
// pull and compiled at search.
func TestNewRegistry_KeepsThePredicateOnlyAtSearch(t *testing.T) {
	t.Parallel()

	// The verdict the policy reads has to name a configured evaluation
	// policy, which is checked as the expression compiles.
	evaluations := policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        "trusted",
		Expressions: []string{`record.name != ""`},
	}}

	registry, err := NewRegistry(context.Background(), evaluations, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPull, policyconfig.OpSearch},
		Name:        "serve-checked-records",
		Expressions: []string{`verdicts["cel:trusted"].compliant_and_current`},
	}}, "", WithVerdicts(&fakeVerdicts{}, func() types.PolicyEnforcement { return types.PolicyEnforcement{} }))
	require.NoError(t, err)

	require.Len(t, registry.byOp[policyconfig.OpSearch], 1)
	assert.Equal(t, searchTerm("cel:trusted"), registry.byOp[policyconfig.OpSearch][0].predicate)

	require.Len(t, registry.byOp[policyconfig.OpPull], 1)
	assert.Nil(t, registry.byOp[policyconfig.OpPull][0].predicate,
		"a policy at pull is evaluated, so it has no predicate to carry")
}
