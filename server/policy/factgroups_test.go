// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"testing"

	"github.com/agntcy/dir/server/types"
	opaast "github.com/open-policy-agent/opa/v1/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func celGroupsOf(t *testing.T, expr string) []string {
	t.Helper()

	env, err := celEnv(layerEvaluation)
	require.NoError(t, err)

	ast, iss := env.Compile(expr)
	require.NoError(t, iss.Err())

	return celFactGroups(ast)
}

func opaGroupsOf(t *testing.T, src string) []string {
	t.Helper()

	mod, err := opaast.ParseModuleWithOpts("test.rego", src, opaast.ParserOptions{RegoVersion: opaast.RegoV1})
	require.NoError(t, err)

	return opaFactGroups(mod)
}

// rego wraps a rule body, so that a test reads as the condition it is about.
func regoRule(body string) string {
	return "package test\n\nallow if {\n\t" + body + "\n}\n"
}

func TestCELFactGroups_ReadsTheGroupsSelected(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		expr string
		want []string
	}{
		"one group": {
			expr: `meta.scan.max_severity == Severity.SEVERITY_NONE`,
			want: []string{"scan"},
		},
		"two groups": {
			expr: `meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED && meta.identity.owner.subject != ""`,
			want: []string{"identity", "signature"},
		},
		"the same group twice is one dependency": {
			expr: `meta.scan.is_safe && meta.scan.finding_count == 0u`,
			want: []string{"scan"},
		},
		"a presence test is a read": {
			expr: `has(meta.scan) && meta.scan.is_safe`,
			want: []string{"scan"},
		},
		"nested selects name the group they start from": {
			expr: `meta.identity.owner.subject.endsWith(".example.com")`,
			want: []string{"identity"},
		},
		"a group inside a collection is still read": {
			expr: `[meta.scan.finding_count, 1u].exists(n, n > 0u)`,
			want: []string{"scan"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, celGroupsOf(t, tc.expr))
		})
	}
}

// A policy that reads no metadata is the one this is all for: nothing a
// scanner or verifier writes can change its answer, so nothing should
// invalidate its verdict.
func TestCELFactGroups_RecordOnlyPolicyDependsOnNothing(t *testing.T) {
	t.Parallel()

	for name, expr := range map[string]string{
		"a record field": `record.name != ""`,
		"the CID":        `record_cid.startsWith("bae")`,
		"indexed_at":     `indexed_at.hasValue()`,
		"no inputs":      `1 == 1`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Empty(t, celGroupsOf(t, expr))
		})
	}
}

// Reaching meta without naming a field leaves no group to read, and guessing
// would mean a verdict that is never invalidated.
func TestCELFactGroups_MetaReachedWholeDependsOnEverything(t *testing.T) {
	t.Parallel()

	for name, expr := range map[string]string{
		"compared whole":           `meta == meta`,
		"tested for presence":      `meta != null`,
		"mixed with a named group": `meta.scan.is_safe && meta == meta`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, types.FactGroups(), celGroupsOf(t, expr))
		})
	}
}

func TestOPAFactGroups_ReadsTheGroupsReferenced(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		src  string
		want []string
	}{
		"one group": {
			src:  regoRule(`input.meta.scan.is_safe`),
			want: []string{"scan"},
		},
		"two groups": {
			src:  regoRule("input.meta.signature.status == \"SIGNATURE_STATUS_VERIFIED\"\n\tinput.meta.identity.owner.status == \"CLAIM_STATUS_VERIFIED\""),
			want: []string{"identity", "signature"},
		},
		"record only": {
			src:  regoRule(`input.record.name != ""`),
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, opaGroupsOf(t, tc.src))
		})
	}
}

// A Rego policy is a module, not an expression: the rule that is queried can
// reach a fact through any other rule in the file, and a dependency read
// through a helper is a dependency.
func TestOPAFactGroups_CoversEveryRuleInTheModule(t *testing.T) {
	t.Parallel()

	src := `package test

allow if {
	input.record.name != ""
	signed
}

signed if input.meta.signature.status == "SIGNATURE_STATUS_VERIFIED"
`

	assert.Equal(t, []string{"signature"}, opaGroupsOf(t, src))
}

func TestOPAFactGroups_UnreadableReferenceDependsOnEverything(t *testing.T) {
	t.Parallel()

	for name, src := range map[string]string{
		"the group is a variable": "package test\n\ngroups := {\"scan\"}\n\nallow if {\n\tsome g in groups\n\tinput.meta[g]\n}\n",
		"meta reached whole":      regoRule(`count(object.keys(input.meta)) > 0`),
		"meta bound to a name":    regoRule("m := input.meta\n\tm.scan.is_safe"),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, types.FactGroups(), opaGroupsOf(t, src))
		})
	}
}

// A policy's dependencies have to reach the evaluator, because the reconciler
// registers them from there alongside the version.
func TestEvaluator_ReportsWhatThePolicyReads(t *testing.T) {
	t.Parallel()

	dir := writePolicy(t, "scan-clean.rego", `package scanclean

allow if input.meta.scan.is_safe
`)

	policies := policiesOf(t, dir,
		opaPolicy("scan-clean.rego"),
		celPolicy("signed", `meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`),
		celPolicy("has-description", `record.description != ""`),
	)

	require.Len(t, policies, 3)
	assert.Equal(t, []string{types.FactGroupScan}, policies[0].FactGroups())
	assert.Equal(t, []string{types.FactGroupSignature}, policies[1].FactGroups())
	assert.Empty(t, policies[2].FactGroups(), "a record-only policy reads no metadata")
}

// Expressions are conjoined into one verdict, so the policy reads whatever any
// of them reads.
func TestEvaluator_UnionsTheGroupsOfEveryExpression(t *testing.T) {
	t.Parallel()

	policies := policiesOf(t, t.TempDir(), celPolicy("mixed",
		`meta.scan.is_safe`,
		`record.name != ""`,
		`meta.identity.owner.status == ClaimStatus.CLAIM_STATUS_VERIFIED`,
	))

	require.Len(t, policies, 1)
	assert.Equal(t, []string{types.FactGroupIdentity, types.FactGroupScan}, policies[0].FactGroups())
}

// Both engines answer in the same vocabulary, so that one dependency set can
// be stored for a policy whichever provider wrote it.
func TestFactGroups_BothEnginesAgree(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		celGroupsOf(t, `meta.scan.is_safe && meta.identity.identity.status == ClaimStatus.CLAIM_STATUS_VERIFIED`),
		opaGroupsOf(t, regoRule("input.meta.scan.is_safe\n\tinput.meta.identity.identity.status == \"CLAIM_STATUS_VERIFIED\"")),
	)
}
