// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"testing"

	opaast "github.com/open-policy-agent/opa/v1/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func celVerdictsOf(t *testing.T, expr string) verdictRefs {
	t.Helper()

	env, err := celEnv(layerOperation)
	require.NoError(t, err)

	ast, iss := env.Compile(expr)
	require.NoError(t, iss.Err())

	return celVerdictKeys(ast)
}

func opaVerdictsOf(t *testing.T, src string) verdictRefs {
	t.Helper()

	mod, err := opaast.ParseModuleWithOpts("test.rego", src, opaast.ParserOptions{RegoVersion: opaast.RegoV1})
	require.NoError(t, err)

	return opaVerdictKeys(mod)
}

func TestCELVerdictKeys_ReadsTheKeysIndexed(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		expr string
		want []string
	}{
		"one verdict": {
			expr: `verdicts["cel:trusted"].compliant_and_current`,
			want: []string{"cel:trusted"},
		},
		"two verdicts": {
			expr: `verdicts["cel:trusted"].compliant_and_current && verdicts["opa:licensed"].compliant_and_current`,
			want: []string{"cel:trusted", "opa:licensed"},
		},
		"the same verdict twice is one key": {
			expr: `verdicts["cel:trusted"].compliant && verdicts["cel:trusted"].status == VerdictStatus.VERDICT_STATUS_EVALUATED`,
			want: []string{"cel:trusted"},
		},
		"reading the status rather than the derived flag still names the key": {
			expr: `verdicts["cel:trusted"].status != VerdictStatus.VERDICT_STATUS_FAILED`,
			want: []string{"cel:trusted"},
		},
		"a policy that reads no verdict names none": {
			expr: `record.name != "" && context.operation == Operation.OPERATION_PULL`,
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			refs := celVerdictsOf(t, tc.expr)
			assert.False(t, refs.all, "the keys are readable")
			assert.Equal(t, tc.want, refs.keys)
		})
	}
}

// A key that is only known while the policy runs is no key at all. Reporting
// none would let the policy enforce before the verdict it reads exists, so
// every one of these has to widen to all.
func TestCELVerdictKeys_AnUnreadableKeyMeansEveryVerdict(t *testing.T) {
	t.Parallel()

	for name, expr := range map[string]string{
		"indexed by a computed key":  `verdicts[record_cid].compliant_and_current`,
		"compared whole":             `size(verdicts) > 0`,
		"iterated":                   `verdicts.all(k, verdicts[k].compliant_and_current)`,
		"one key read and one not":   `verdicts["cel:trusted"].compliant && size(verdicts) > 1`,
		"indexed by a concatenation": `verdicts["cel:" + record_cid].compliant`,
		// A policy ID is provider:name, so a field selection cannot name one.
		"selected as a field": `verdicts.trusted.compliant_and_current`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, celVerdictsOf(t, expr).all, "expected every verdict for %s", expr)
		})
	}
}

func TestOPAVerdictKeys_ReadsTheKeysReferenced(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		src  string
		want []string
	}{
		"one verdict": {
			src:  regoRule(`input.verdicts["cel:trusted"].compliant_and_current`),
			want: []string{"cel:trusted"},
		},
		"two verdicts": {
			src: regoRule(`input.verdicts["cel:trusted"].compliant_and_current
	input.verdicts["opa:licensed"].compliant_and_current`),
			want: []string{"cel:trusted", "opa:licensed"},
		},
		"a policy that reads no verdict names none": {
			src:  regoRule(`input.record.name != ""`),
			want: nil,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			refs := opaVerdictsOf(t, tc.src)
			assert.False(t, refs.all, "the keys are readable")
			assert.Equal(t, tc.want, refs.keys)
		})
	}
}

// A verdict read through a helper rule is read by the policy just the same,
// so every rule in the module counts rather than only the queried one.
func TestOPAVerdictKeys_CoversEveryRuleInTheModule(t *testing.T) {
	t.Parallel()

	src := `package test

allow if {
	trusted
}

trusted if {
	input.verdicts["cel:trusted"].compliant_and_current
}
`

	refs := opaVerdictsOf(t, src)
	assert.False(t, refs.all)
	assert.Equal(t, []string{"cel:trusted"}, refs.keys)
}

func TestOPAVerdictKeys_AnUnreadableReferenceMeansEveryVerdict(t *testing.T) {
	t.Parallel()

	for name, src := range map[string]string{
		"indexed by a variable": regoRule(`input.verdicts[input.record_cid].compliant_and_current`),
		"reached whole":         regoRule(`count(input.verdicts) > 0`),
		"iterated": `package test

allow if {
	some key
	input.verdicts[key].compliant_and_current
}
`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.True(t, opaVerdictsOf(t, src).all, "expected every verdict for %s", src)
		})
	}
}

// The two engines answer the same question, so a policy written both ways has
// to report the same keys. Nothing makes that true but writing both.
func TestVerdictKeys_BothEnginesAgree(t *testing.T) {
	t.Parallel()

	cel := celVerdictsOf(t,
		`verdicts["cel:trusted"].compliant_and_current && verdicts["opa:licensed"].compliant`)
	opa := opaVerdictsOf(t, regoRule(`input.verdicts["cel:trusted"].compliant_and_current
	input.verdicts["opa:licensed"].compliant`))

	assert.Equal(t, cel, opa)
}
