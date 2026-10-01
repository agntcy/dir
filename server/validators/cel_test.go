// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/stretchr/testify/require"
)

func celConfig(exprs ...string) map[string]any {
	return map[string]any{validatorsconfig.ConfigKeyExpressions: exprs}
}

func TestNewCELValidator_RejectsBadExpression(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig("this is not cel"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "compile CEL expressions[0]")
}

func TestNewCELValidator_RejectsNonBool(t *testing.T) {
	t.Parallel()

	_, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig("record.name"),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must evaluate to bool")
}

func TestCELValidator_AcceptsMatchingRecord(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig(`record.name != ""`, `record.version != ""`),
	})
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent", Version: "1.0.0"})
	ok, msgs, warnings, err := v.ValidateRecord(context.Background(), record.GetData())
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
	require.Empty(t, warnings)
}

func TestCELValidator_FailsIfAnyExpressionDoesNotMatch(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig(
			`record.name != ""`,
			`record.name.startsWith("cisco.com/")`,
			`record.name.endsWith("/allowed")`,
		),
	})
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "other.com/agent"})
	ok, msgs, _, err := v.ValidateRecord(context.Background(), record.GetData())
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, []string{
		`CEL expression evaluated to false: record.name.startsWith("cisco.com/")`,
		`CEL expression evaluated to false: record.name.endsWith("/allowed")`,
	}, msgs)
}

func TestCELValidator_EvalErrorIsValidationFailure(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig(`record.missing_field == "x"`),
	})
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	ok, msgs, _, err := v.ValidateRecord(context.Background(), record.GetData())
	require.NoError(t, err)
	require.False(t, ok)
	require.NotEmpty(t, msgs)
}

func TestCELValidator_NilData(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig(`true`),
	})
	require.NoError(t, err)

	ok, msgs, _, err := v.ValidateRecord(context.Background(), nil)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, []string{"record data is nil"}, msgs)
}

func TestCELValidator_CanceledContext(t *testing.T) {
	t.Parallel()

	v, err := newCELValidator(validatorsconfig.Validator{
		Config: celConfig(`true`),
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok, msgs, _, err := v.ValidateRecord(ctx, nil)
	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, ok)
	require.Empty(t, msgs)
}

func TestNewRegistry_CEL(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), validatorsconfig.Config{{
		Provider: validatorsconfig.ProviderCEL,
		Ops:      []string{validatorsconfig.OpPush},
		Config:   celConfig(`record.name != ""`),
	}}, "")
	require.NoError(t, err)
	require.Len(t, reg.For(validatorsconfig.OpPush), 1)

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})
	ok, msgs, err := reg.Run(context.Background(), validatorsconfig.OpPush, record)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
}
