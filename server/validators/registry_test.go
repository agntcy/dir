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
	"google.golang.org/protobuf/types/known/structpb"
)

func TestNewRegistry_Empty(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(nil)
	require.NoError(t, err)
	require.Empty(t, reg.For(validatorsconfig.OpPush))
	require.Empty(t, reg.For(validatorsconfig.OpIndex))
}

func TestNewRegistry_RejectsEmptySchemaURL(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(validatorsconfig.Config{{
		Provider: validatorsconfig.ProviderOASF,
		Ops:      []string{validatorsconfig.OpPush},
	}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "schema_url is required")
}

func TestNewRegistry_MultiplePerOp(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(validatorsconfig.Config{
		{
			Provider:  validatorsconfig.ProviderOASF,
			SchemaURL: "https://schema.example.com/a",
			Ops:       []string{validatorsconfig.OpPush},
		},
		{
			Provider:  validatorsconfig.ProviderOASF,
			SchemaURL: "https://schema.example.com/b",
			Ops:       []string{validatorsconfig.OpPush, validatorsconfig.OpIndex},
		},
	})
	require.NoError(t, err)
	require.Len(t, reg.For(validatorsconfig.OpPush), 2)
	require.Len(t, reg.For(validatorsconfig.OpIndex), 1)
	require.Empty(t, reg.For(validatorsconfig.OpAutosync))
}

func TestNewRegistry_ReusesSameURL(t *testing.T) {
	t.Parallel()

	reg, err := NewRegistry(validatorsconfig.Config{{
		Provider:  validatorsconfig.ProviderOASF,
		SchemaURL: "https://schema.oasf.outshift.com",
		Ops:       []string{validatorsconfig.OpPush, validatorsconfig.OpIndex},
	}})
	require.NoError(t, err)

	push := reg.For(validatorsconfig.OpPush)
	index := reg.For(validatorsconfig.OpIndex)

	require.Len(t, push, 1)
	require.Len(t, index, 1)
	require.Equal(t, push[0], index[0])
	require.Empty(t, reg.For(validatorsconfig.OpAutosync))
}

type fakeValidator struct {
	valid bool
	msgs  []string
}

func (f fakeValidator) ValidateRecord(context.Context, *structpb.Struct) (bool, []string, []string, error) {
	return f.valid, f.msgs, nil, nil
}

func TestRun_EmptyIsValid(t *testing.T) {
	t.Parallel()

	var reg *Registry

	ok, msgs, err := reg.Run(context.Background(), validatorsconfig.OpPush, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
}

func TestRun_CollectsFailures(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "test"})
	reg := With(validatorsconfig.OpPush,
		fakeValidator{valid: true},
		fakeValidator{valid: false, msgs: []string{"first"}},
		fakeValidator{valid: false, msgs: []string{"second"}},
	)

	ok, msgs, err := reg.Run(context.Background(), validatorsconfig.OpPush, record)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, []string{"ERROR: first", "ERROR: second"}, msgs)
}
