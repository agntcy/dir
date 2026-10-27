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
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewCELValidator_RejectsBadExpression(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{"this is not cel"}, layerEvaluation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "compile CEL expressions[0]")
}

func TestNewCELValidator_RejectsNonBool(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator([]string{"record.name"}, layerEvaluation)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must evaluate to bool")
}

func TestCELValidator_AcceptsMatchingRecord(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{`record.name != ""`, `record.version != ""`}, layerEvaluation)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent", Version: "1.0.0"})
	ok, msgs, warnings, err := v.ValidateEvaluation(context.Background(), recordState(record))
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
	require.Empty(t, warnings)
}

func TestCELValidator_FailsIfAnyExpressionDoesNotMatch(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{
		`record.name != ""`,
		`record.name.startsWith("cisco.com/")`,
		`record.name.endsWith("/allowed")`,
	}, layerEvaluation)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "other.com/agent"})
	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), recordState(record))
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, []string{
		`CEL expression evaluated to false: record.name.startsWith("cisco.com/")`,
		`CEL expression evaluated to false: record.name.endsWith("/allowed")`,
	}, msgs)
}

func TestCELValidator_EvalErrorIsValidationFailure(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{`record.missing_field == "x"`}, layerEvaluation)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), recordState(record))
	require.NoError(t, err)
	require.False(t, ok)
	require.NotEmpty(t, msgs)
}

// The typed environment must not change how a record expression reads.
// Policies are content-hashed, so an expression that has to be rewritten gets
// a new version and loses the verdicts recorded under the old one.
func TestCELValidator_RecordStaysDynamic(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{
		Name:    "cisco.com/agent",
		Version: "1.0.0",
		Locators: []*typesv1alpha1.Locator{
			{Type: "docker-image", Url: "ghcr.io/example/agent:1.0.0"},
		},
	})

	v, err := newCELValidator([]string{
		`record.name.startsWith("cisco.com/")`,
		`has(record.locators)`,
		`size(record.locators) == 1`,
		`record.locators[0].type == "docker-image"`,
		`record.locators.all(l, l.url != "")`,
		`!has(record.unknown_field)`,
	}, layerEvaluation)
	require.NoError(t, err)

	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), recordState(record))
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
}

// A misspelled record field still compiles, because the record has no
// compile-time schema, and still fails at evaluation.
func TestCELValidator_UnknownRecordFieldStillCompiles(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{`record.locatrs.size() > 0`}, layerEvaluation)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), recordState(record))
	require.NoError(t, err)
	require.False(t, ok)
	require.NotEmpty(t, msgs)
}

func TestCELValidator_NilData(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{`true`}, layerEvaluation)
	require.NoError(t, err)

	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), nil)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, []string{"record data is nil"}, msgs)
}

func TestCELValidator_CanceledContext(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{`true`}, layerEvaluation)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok, msgs, _, err := v.ValidateEvaluation(ctx, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, ok)
	require.Empty(t, msgs)
}

func TestNewRegistry_CEL(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider:    policyconfig.ProviderCEL,
		Ops:         []string{policyconfig.OpPush},
		Name:        "require-name",
		Expressions: []string{`record.name != ""`},
	}}, "")
	require.NoError(t, err)
	require.Len(t, reg.For(policyconfig.OpPush), 1)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush, record)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
}

func TestCELValidator_ReadsRecordState(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{
		`record.name == "cisco.com/agent"`,
		`record_cid == "bafy"`,
		`indexed_at.hasValue()`,
		`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_VERIFIED`,
	}, layerEvaluation)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	state := recordState(record)
	state.RecordCid = "bafy"
	state.IndexedAt = timestamppb.Now()
	state.Meta = &policyv1.RecordMeta{
		Signature: &policyv1.SignatureFacts{
			Status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
		},
	}

	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), state)
	require.NoError(t, err)
	require.True(t, ok, msgs)
	require.Empty(t, msgs)
}

// The operation layer reads the record, the circumstances of the call, and the
// instant it is being decided at.
func TestCELValidator_ReadsOperationInput(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{
		`record.name == "cisco.com/agent"`,
		`record_cid == "bafy"`,
		`context.operation == Operation.OPERATION_PUSH`,
		`now > timestamp("2020-01-01T00:00:00Z")`,
	}, layerOperation)
	require.NoError(t, err)

	input := operationInput(policyconfig.OpPush,
		corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"}))
	input.RecordCid = "bafy"

	ok, msgs, warnings, err := v.ValidateOperation(context.Background(), input)
	require.NoError(t, err)
	require.True(t, ok, msgs)
	require.Empty(t, msgs)
	require.Empty(t, warnings)
}

// One policy configured for several operations can hold each to its own rule,
// which is what reading context is for. This is the example in the records
// validation docs.
func TestCELValidator_SamePolicyDecidesPerOperation(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{
		`context.operation != Operation.OPERATION_AUTOSYNC || has(record.locators)`,
	}, layerOperation)
	require.NoError(t, err)

	bare := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	located := corev1.New(&typesv1alpha1.Record{
		Name:     "cisco.com/agent",
		Locators: []*typesv1alpha1.Locator{{Type: "docker-image", Url: "ghcr.io/example/agent:1"}},
	})

	tests := []struct {
		name   string
		op     string
		record *corev1.Record
		want   bool
	}{
		{"push is not held to it", policyconfig.OpPush, bare, true},
		{"autosync without locators is rejected", policyconfig.OpAutosync, bare, false},
		{"autosync with locators is accepted", policyconfig.OpAutosync, located, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ok, msgs, _, err := v.ValidateOperation(context.Background(),
				operationInput(tc.op, tc.record))
			require.NoError(t, err)
			require.Equal(t, tc.want, ok, msgs)
		})
	}
}

// Nothing populates source_peer_id yet, so it reads as absent rather than as
// the empty string, and a policy can tell the two apart once it is populated.
func TestCELValidator_SourcePeerIDReadsAbsent(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{`!has(context.source_peer_id)`}, layerOperation)
	require.NoError(t, err)

	ok, msgs, _, err := v.ValidateOperation(context.Background(), operationInput(
		policyconfig.OpPush, corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})))
	require.NoError(t, err)
	require.True(t, ok, msgs)
}

func TestCELValidator_UnsetFactsStayAbsent(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator([]string{
		`!indexed_at.hasValue()`,
		`meta.signature.status == SignatureStatus.SIGNATURE_STATUS_UNSPECIFIED`,
	}, layerEvaluation)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), recordState(record))
	require.NoError(t, err)
	require.True(t, ok, msgs)
	require.Empty(t, msgs)
}
