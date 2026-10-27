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
)

func TestNewRegistry_Empty(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(context.Background(), nil, nil, "")
	require.NoError(t, err)
	require.Empty(t, reg.For(policyconfig.OpPush))
	require.Empty(t, reg.For(policyconfig.OpIndex))
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
	require.Len(t, reg.For(policyconfig.OpPush), 2)
	require.Len(t, reg.For(policyconfig.OpIndex), 1)
	require.Empty(t, reg.For(policyconfig.OpAutosync))
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
	require.Len(t, reg.For(policyconfig.OpPush), 1)
	require.Len(t, reg.For(policyconfig.OpIndex), 1)
	require.Empty(t, reg.For(policyconfig.OpAutosync))
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
