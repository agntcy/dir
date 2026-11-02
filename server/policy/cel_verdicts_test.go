// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	"cel.dev/cel-go/cel"
	"cel.dev/cel-go/ext"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/stretchr/testify/require"
)

// operationEnvWithVerdicts is the operation environment plus the verdict map,
// which production does not declare until a read can fill it. The key check
// is written against that map, so the tests declare it here.
func operationEnvWithVerdicts(t *testing.T) *cel.Env {
	t.Helper()

	env, err := newOperationEnv()
	require.NoError(t, err)

	env, err = env.Extend(cel.Variable(celVarVerdicts, cel.MapType(
		cel.StringType,
		cel.ObjectType(policyContainer+".Verdict"),
	)))
	require.NoError(t, err)

	return env
}

func compileVerdicts(t *testing.T, exprs []string, ids []string) error {
	t.Helper()

	_, err := compileCEL(exprs, operationEnvWithVerdicts(t), ids)

	return err
}

func TestValidateAst_AcceptsLiteralKey(t *testing.T) {
	t.Parallel()

	ids := []string{"cel:has-provenance", "opa:require-license"}

	err := compileVerdicts(t, []string{
		`verdicts["cel:has-provenance"].compliant`,
		`verdicts["opa:require-license"].compliant_and_current`,
		`verdicts["cel:has-provenance"].compliant && verdicts["opa:require-license"].compliant_and_current`,
		`"cel:has-provenance" in verdicts`,
		`has(verdicts["cel:has-provenance"].evaluated_at)`,
		`verdicts[?"cel:has-provenance"].hasValue()`,
	}, ids)
	require.NoError(t, err)
}

func TestValidateAst_RejectsUnknownKey(t *testing.T) {
	t.Parallel()

	err := compileVerdicts(t, []string{
		`verdicts["cel:has-provenence"].compliant_and_current`,
	}, []string{"cel:has-provenance", "opa:require-license"})
	require.ErrorContains(t, err, `CEL expressions[0]: verdict key "cel:has-provenence"`)
	require.ErrorContains(t, err, "configured: cel:has-provenance, opa:require-license")

	err = compileVerdicts(t, []string{
		`verdicts["cel:has-provenance"].compliant`,
		`verdicts["cel:missing"].compliant`,
	}, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, "CEL expressions[1]")
	require.ErrorContains(t, err, `verdict key "cel:missing"`)
	require.ErrorContains(t, err, "configured: cel:has-provenance")

	err = compileVerdicts(t, []string{`"cel:missing" in verdicts`}, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, `verdict key "cel:missing"`)

	err = compileVerdicts(t, []string{`verdicts["cel:has-provenance"].compliant`}, nil)
	require.ErrorContains(t, err, `verdict key "cel:has-provenance"`)
	require.ErrorContains(t, err, "configured: none")

	err = compileVerdicts(t, []string{`verdicts[?"cel:missing"].hasValue()`}, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, `verdict key "cel:missing"`)
}

func TestValidateAst_RejectsComputedKey(t *testing.T) {
	t.Parallel()

	ids := []string{"cel:has-provenance"}

	for _, expr := range []string{
		`verdicts["cel:" + "has-provenance"].compliant`,
		`verdicts[record.name].compliant`,
		`record.name in verdicts`,
	} {
		err := compileVerdicts(t, []string{expr}, ids)
		require.ErrorContains(t, err, "not a literal string", expr)
	}
}

func TestValidateAst_RejectsHiddenUse(t *testing.T) {
	t.Parallel()

	ids := []string{"cel:has-provenance"}

	for _, expr := range []string{
		`verdicts.all(k, verdicts[k].compliant_and_current)`,
		`size(verdicts) == 1`,
	} {
		err := compileVerdicts(t, []string{expr}, ids)
		require.ErrorContains(t, err, "without a literal key", expr)
	}
}

// cel.bind is not part of the policy environment. When it is enabled, binding
// the map still has to be refused: the alias hides the key.
func TestValidateAst_RejectsBind(t *testing.T) {
	t.Parallel()

	env, err := operationEnvWithVerdicts(t).Extend(ext.Bindings())
	require.NoError(t, err)

	_, err = compileCEL([]string{
		`cel.bind(v, verdicts, v["cel:has-provenance"].compliant)`,
	}, env, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, "without a literal key")
}

func TestValidateAst_FieldTypoIsACompileError(t *testing.T) {
	t.Parallel()

	err := compileVerdicts(t, []string{
		`verdicts["cel:has-provenance"].compliant_and_currnet`,
	}, []string{"cel:has-provenance"})
	require.ErrorContains(t, err, "compile CEL expressions[0]")
	require.NotContains(t, err.Error(), "verdict key")
}

// The operation environment declares verdicts, so a key is checked against the
// configured evaluation policies rather than failing as an unknown variable.
// newCELValidator is given none, and the key names none.
func TestOperationEnv_RejectsVerdictKeyWithNoEvaluations(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{`verdicts["cel:has-provenance"].compliant`}, layerOperation)
	require.ErrorContains(t, err, `verdict key "cel:has-provenance"`)
	require.ErrorContains(t, err, "configured: none")
}

func TestNewRegistry_NamesPolicyOnVerdictCompileFailure(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(context.Background(), policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        "has-provenance",
		Expressions: []string{`true`},
	}}, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush},
		Name:        "serve",
		Expressions: []string{`verdicts["cel:has-provenence"].compliant`},
	}}, "")
	require.ErrorContains(t, err, `operations[0]: policy "cel:serve"`)
	require.ErrorContains(t, err, `verdict key "cel:has-provenence"`)
	require.ErrorContains(t, err, "configured: cel:has-provenance")
}

func TestNewRegistry_CELWithEvaluations(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), policyconfig.Evaluations{{
		Provider:    policyconfig.ProviderCEL,
		Name:        "has-provenance",
		Expressions: []string{`true`},
	}}, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush},
		Name:        "require-name",
		Expressions: []string{`record.name != ""`},
	}}, "")
	require.NoError(t, err)
	require.Equal(t, []string{"cel:has-provenance"}, reg.EvaluationIDs())
	require.Equal(t, []string{"cel:require-name"}, reg.OperationIDs(policyconfig.OpPush))
}
