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

// Every op a policy can be configured at has to name an operation. An op
// missing from the map would read as OPERATION_UNSPECIFIED, which a policy
// comparing context.operation would answer confidently and wrongly.
func TestOperationInput_NamesEveryConfigurableOp(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	for _, op := range policyconfig.KnownOps() {
		input := operationInput(op, record)
		require.NotNil(t, input)
		require.NotEqual(t, policyv1.Operation_OPERATION_UNSPECIFIED,
			input.GetContext().GetOperation(), "op %q has no operation", op)
	}
}

// An absent peer and a peer with an empty name have to read differently, so
// an empty id leaves the field unset rather than reporting a peer called "".
func TestFromPeer_EmptyLeavesItUnset(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	set := operationInput(policyconfig.OpAutosync, record, FromPeer("12D3KooWSomePeer"))
	require.Equal(t, "12D3KooWSomePeer", set.GetContext().GetSourcePeerId())
	require.NotNil(t, set.GetContext().SourcePeerId)

	unset := operationInput(policyconfig.OpAutosync, record, FromPeer(""))
	require.Nil(t, unset.GetContext().SourcePeerId)
}

// A policy reads context and now as values, so an input missing either fails
// the validator rather than letting the policy decide on 1970.
func TestCheckOperationInput_RequiresContextAndNow(t *testing.T) {
	t.Parallel()

	record := corev1.New(&typesv1alpha1.Record{Name: "cisco.com/agent"})

	tests := []struct {
		name  string
		input *policyv1.OperationInput
		want  string
	}{
		{
			name:  "no context",
			input: &policyv1.OperationInput{Record: record.GetData(), Now: timestamppb.Now()},
			want:  "no context",
		},
		{
			name: "no now",
			input: &policyv1.OperationInput{
				Record:  record.GetData(),
				Context: &policyv1.OperationContext{Operation: policyv1.Operation_OPERATION_PUSH},
			},
			want: "no now",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorContains(t, checkOperationInput(tc.input), tc.want)

			// Each provider reports it the same way, because each hands the
			// missing value to its policy.
			cel, err := newCELValidator([]string{`true`}, layerOperation)
			require.NoError(t, err)

			ok, msgs, _, err := cel.ValidateOperation(context.Background(), tc.input)
			require.ErrorContains(t, err, tc.want)
			require.False(t, ok)
			require.Empty(t, msgs, "a failed validator reports nothing about the record")

			opa, err := newOPAValidator(context.Background(),
				writePolicy(t, "allow.rego", allowIfNamed), "allow.rego")
			require.NoError(t, err)

			ok, msgs, _, err = opa.ValidateOperation(context.Background(), tc.input)
			require.ErrorContains(t, err, tc.want)
			require.False(t, ok)
			require.Empty(t, msgs)
		})
	}
}
