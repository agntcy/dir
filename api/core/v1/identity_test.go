// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func newRecordWithAnnotations(t *testing.T, annotations map[string]any) *corev1.Record {
	t.Helper()

	data, err := structpb.NewStruct(map[string]any{"annotations": annotations})
	require.NoError(t, err)

	return &corev1.Record{Data: data}
}

func TestRecord_GetIdentity(t *testing.T) {
	record := newRecordWithAnnotations(t, map[string]any{
		corev1.AnnotationKeyIdentity: "did:web:acme.com:agents:finance",
	})
	require.Equal(t, "did:web:acme.com:agents:finance", record.GetIdentity())

	require.Empty(t, (&corev1.Record{}).GetIdentity())
	require.Empty(t, (*corev1.Record)(nil).GetIdentity())
}

func TestRecord_GetOwner(t *testing.T) {
	record := newRecordWithAnnotations(t, map[string]any{
		corev1.AnnotationKeyOwner: "dns:acme.com",
	})
	require.Equal(t, "dns:acme.com", record.GetOwner())

	require.Empty(t, (&corev1.Record{}).GetOwner())
	require.Empty(t, (*corev1.Record)(nil).GetOwner())
}
