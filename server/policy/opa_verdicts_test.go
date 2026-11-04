// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/stretchr/testify/require"
)

func parseModule(t *testing.T, src string) *ast.Module {
	t.Helper()

	mod, err := ast.ParseModuleWithOpts("test.rego", src, ast.ParserOptions{RegoVersion: ast.RegoV1})
	require.NoError(t, err)

	return mod
}

func TestValidateOPAModule_AcceptsLiteralKey(t *testing.T) {
	t.Parallel()

	ids := []string{"cel:has-provenance", "opa:require-license"}

	for _, body := range []string{
		`input.verdicts["cel:has-provenance"].compliant`,
		`input.verdicts["opa:require-license"].compliant_and_current`,
		`input.verdicts["cel:has-provenance"].compliant
	input.verdicts["opa:require-license"].compliant_and_current`,
		`"cel:has-provenance" in input.verdicts`,
		`input.verdicts["cel:has-provenance"].status == "VERDICT_STATUS_EVALUATED"`,
	} {
		err := validateOPAModule(parseModule(t, regoRule(body)), layerOperation, ids)
		require.NoError(t, err, body)
	}
}

func TestValidateOPAModule_RejectsUnknownKey(t *testing.T) {
	t.Parallel()

	err := validateOPAModule(parseModule(t, regoRule(
		`input.verdicts["cel:has-provenence"].compliant_and_current`,
	)), layerOperation, []string{"cel:has-provenance", "opa:require-license"})
	require.ErrorContains(t, err, `verdict key "cel:has-provenence"`)
	require.ErrorContains(t, err, "configured: cel:has-provenance, opa:require-license")

	err = validateOPAModule(parseModule(t, regoRule(
		`"cel:missing" in input.verdicts`,
	)), layerOperation, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, `verdict key "cel:missing"`)
	require.ErrorContains(t, err, "configured: cel:has-provenance")

	err = validateOPAModule(parseModule(t, regoRule(
		`input.verdicts["cel:has-provenance"].compliant`,
	)), layerOperation, nil)
	require.ErrorContains(t, err, `verdict key "cel:has-provenance"`)
	require.ErrorContains(t, err, "configured: none")
}

func TestValidateOPAModule_RejectsComputedKey(t *testing.T) {
	t.Parallel()

	ids := []string{"cel:has-provenance"}

	for _, body := range []string{
		`input.verdicts[input.record_cid].compliant`,
		`input.verdicts[concat("", ["cel:", "has-provenance"])].compliant`,
		`input.record_cid in input.verdicts`,
		`some key
	input.verdicts[key].compliant_and_current`,
	} {
		err := validateOPAModule(parseModule(t, regoRule(body)), layerOperation, ids)
		require.ErrorContains(t, err, "not a literal string", body)
	}
}

func TestValidateOPAModule_RejectsHiddenUse(t *testing.T) {
	t.Parallel()

	ids := []string{"cel:has-provenance"}

	for _, src := range []string{
		regoRule(`count(input.verdicts) > 0`),
		regoRule(`v := input.verdicts
	v["cel:has-provenance"].compliant`),
		`package test

allow if {
	every _, v in input.verdicts {
		v.compliant_and_current
	}
}
`,
		regoRule(`object.get(input.verdicts, "cel:has-provenance", {}).compliant`),
	} {
		err := validateOPAModule(parseModule(t, src), layerOperation, ids)
		require.ErrorContains(t, err, "without a literal key", src)
	}
}

// A verdict read through a helper rule is the policy's read, so the key is
// checked there too.
func TestValidateOPAModule_CoversEveryRule(t *testing.T) {
	t.Parallel()

	src := `package test

allow if {
	trusted
}

trusted if {
	input.verdicts["cel:missing"].compliant_and_current
}
`

	err := validateOPAModule(parseModule(t, src), layerOperation, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, `verdict key "cel:missing"`)
}

// The evaluation layer has no verdicts to read. A reference would parse, and
// a body that cannot be evaluated would fall through to default allow.
func TestValidateOPAModule_RejectsVerdictsInEvaluation(t *testing.T) {
	t.Parallel()

	err := validateOPAModule(parseModule(t, regoRule(
		`input.verdicts["cel:has-provenance"].compliant`,
	)), layerEvaluation, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, "evaluation policy")
	require.ErrorContains(t, err, "verdicts")

	err = validateOPAModule(parseModule(t, regoRule(`input.record.name != ""`)), layerEvaluation, nil)
	require.NoError(t, err)
}

func TestNewRegistry_OPANamesPolicyOnVerdictFailure(t *testing.T) {
	t.Parallel()

	dir := writePolicy(t, "gate.rego", regoRule(`input.verdicts["cel:has-provenence"].compliant`))

	_, err := NewRegistry(context.Background(), policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        "has-provenance",
		Expressions: []string{`true`},
	}}, policyconfig.Operations{{
		Provider: policyconfig.ProviderOPA,
		Ops:      []string{policyconfig.OpPull},
		Name:     "gate",
		File:     "gate.rego",
	}}, dir)
	require.ErrorContains(t, err, `operations[0]: policy "opa:gate"`)
	require.ErrorContains(t, err, `policy file gate.rego`)
	require.ErrorContains(t, err, `verdict key "cel:has-provenence"`)
	require.ErrorContains(t, err, "configured: cel:has-provenance")
}
