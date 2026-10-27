// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"cel.dev/cel-go/cel"
	corev1 "github.com/agntcy/dir/api/core/v1"
	identityv1 "github.com/agntcy/dir/api/identity/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	securityv1 "github.com/agntcy/dir/api/security/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/stretchr/testify/require"
)

// evalWithMeta compiles expr in the policy environment and evaluates it
// against meta. These tests bind meta directly to exercise the typed half of
// the environment; the validator binds the same variable from RecordState.
func evalWithMeta(t *testing.T, expr string, meta *policyv1.RecordMeta) bool {
	t.Helper()

	env, err := newEvaluationEnv()
	require.NoError(t, err)

	ast, iss := env.Compile(expr)
	require.NoError(t, iss.Err())
	require.Equal(t, cel.BoolType, ast.OutputType())

	prg, err := env.Program(ast)
	require.NoError(t, err)

	out, _, err := prg.Eval(map[string]any{
		celVarRecord: map[string]any{},
		celVarMeta:   meta,
	})
	require.NoError(t, err)

	got, ok := out.Value().(bool)
	require.True(t, ok, "expression returned %T, want bool", out.Value())

	return got
}

// A misspelled typed field must fail when the policy is loaded, which is the
// point of giving CEL a typed environment at all.
func TestPolicyEnv_MisspelledFieldFailsAtLoad(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{`meta.signature.statsu == SignatureStatus.SIGNATURE_STATUS_VERIFIED`}, layerEvaluation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "compile CEL expressions[0]")
	require.Contains(t, err.Error(), "statsu", "the error should name the misspelled field")
}

// A misspelled message is caught the same way.
func TestPolicyEnv_MisspelledGroupFailsAtLoad(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{`meta.signatrue.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`}, layerEvaluation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "signatrue")
}

// An enum constant declared in policy/v1 is written by its short name, which
// is what cel.Container buys.
func TestPolicyEnv_ContainerResolvesPolicyEnums(t *testing.T) {
	t.Parallel()

	meta := &policyv1.RecordMeta{
		Signature: &policyv1.SignatureFacts{
			Status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			Source: policyv1.FactSource_FACT_SOURCE_LOCAL,
		},
	}

	require.True(t, evalWithMeta(t,
		`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`, meta))
	require.True(t, evalWithMeta(t,
		`meta.signature.source == FactSource.FACT_SOURCE_LOCAL`, meta))
	require.False(t, evalWithMeta(t,
		`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_REVOKED`, meta))
}

// Severity, ScannerType and ClaimRole are declared outside policy/v1, so they
// resolve by short name only because they are abbreviated. Without that a
// policy would spell some enums short and others fully qualified.
func TestPolicyEnv_AbbrevsResolveExternalEnums(t *testing.T) {
	t.Parallel()

	meta := &policyv1.RecordMeta{
		Scan: &policyv1.ScanFacts{
			Status:      policyv1.ScanStatus_SCAN_STATUS_COMPLETED,
			MaxSeverity: securityv1.Severity_SEVERITY_HIGH,
			Scanners: []*policyv1.ScannerResult{{
				ScannerType: securityv1.ScannerType_SCANNER_TYPE_MCP,
				MaxSeverity: securityv1.Severity_SEVERITY_HIGH,
			}},
		},
		Identity: &policyv1.IdentityFacts{
			Owner: &policyv1.ClaimFacts{
				Role:   identityv1.ClaimRole_CLAIM_ROLE_OWNER,
				Status: policyv1.ClaimStatus_CLAIM_STATUS_VERIFIED,
			},
		},
	}

	require.True(t, evalWithMeta(t,
		`meta.scan.max_severity == Severity.SEVERITY_HIGH`, meta))
	require.True(t, evalWithMeta(t,
		`meta.scan.scanners[0].scanner_type == ScannerType.SCANNER_TYPE_MCP`, meta))
	require.True(t, evalWithMeta(t,
		`meta.identity.owner.role == ClaimRole.CLAIM_ROLE_OWNER`, meta))
}

// Severity is declared ascending, so a threshold is an ordinary comparison.
func TestPolicyEnv_SeverityThreshold(t *testing.T) {
	t.Parallel()

	const expr = `meta.scan.max_severity >= Severity.SEVERITY_MEDIUM`

	tests := []struct {
		severity securityv1.Severity
		want     bool
	}{
		{securityv1.Severity_SEVERITY_CRITICAL, true},
		{securityv1.Severity_SEVERITY_HIGH, true},
		{securityv1.Severity_SEVERITY_MEDIUM, true},
		{securityv1.Severity_SEVERITY_LOW, false},
		{securityv1.Severity_SEVERITY_NONE, false},
		// An unscanned record must fail the threshold rather than match it.
		{securityv1.Severity_SEVERITY_UNSPECIFIED, false},
	}

	for _, tc := range tests {
		meta := &policyv1.RecordMeta{Scan: &policyv1.ScanFacts{MaxSeverity: tc.severity}}
		require.Equal(t, tc.want, evalWithMeta(t, expr, meta), "severity %s", tc.severity)
	}
}

// An absent fact group reads as its zero value rather than erroring, so a
// policy requiring something positive rejects the record instead of failing
// open. This is what the validator binds today for every record.
func TestPolicyEnv_EmptyMetaFailsPositiveRequirements(t *testing.T) {
	t.Parallel()

	empty := &policyv1.RecordMeta{}

	require.False(t, evalWithMeta(t,
		`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`, empty))
	require.False(t, evalWithMeta(t,
		`meta.scan.status == ScanStatus.SCAN_STATUS_COMPLETED`, empty))
	require.True(t, evalWithMeta(t,
		`meta.scan.max_severity == Severity.SEVERITY_UNSPECIFIED`, empty))
}

// An evaluation policy is compiled without the operation clock or verdicts.
// Those are what RecordState is shaped to withhold.
func TestRecordEnv_RejectsOperationFields(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{`now > timestamp("2020-01-01T00:00:00Z")`}, layerEvaluation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "now")

	_, err = newCELValidator([]string{`verdicts["cel:x"].compliant`}, layerEvaluation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verdicts")
}

// An operation policy is compiled without the facts. Reaching one is what a
// verdict is for, so a policy that reads meta here is not held back at run
// time, it does not load at all.
func TestOperationEnv_RejectsEvaluationFields(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{
		`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`,
	}, layerOperation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "meta")

	_, err = newCELValidator([]string{`indexed_at.hasValue()`}, layerOperation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "indexed_at")
}

// A policy is compiled for one layer and can only answer for that layer, so
// asking it the other question fails the validator rather than the record.
func TestCELValidator_RejectsWrongLayer(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	op, err := newCELValidator([]string{`true`}, layerOperation)
	require.NoError(t, err)

	ok, msgs, _, err := op.ValidateEvaluation(context.Background(), recordState(record))
	require.ErrorContains(t, err, "cannot produce a verdict")
	require.False(t, ok)
	require.Empty(t, msgs, "a failed validator reports nothing about the record")

	eval, err := newCELValidator([]string{`true`}, layerEvaluation)
	require.NoError(t, err)

	ok, msgs, _, err = eval.ValidateOperation(context.Background(),
		operationInput(policyconfig.OpPush, record))
	require.ErrorContains(t, err, "cannot decide an operation")
	require.False(t, ok)
	require.Empty(t, msgs)
}
