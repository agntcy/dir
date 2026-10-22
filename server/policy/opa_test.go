// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	policyv1 "github.com/agntcy/dir/api/policy/v1"
	policyconfig "github.com/agntcy/dir/server/policy/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

const allowIfNamed = `
package dir.policy.require_name

default allow := false

allow if {
	input.record.name != ""
}
`

func writePolicy(t *testing.T, filename, src string) string {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, filename), []byte(src), 0o600))

	return dir
}

func TestNewOPAValidator_AllowAndDeny(t *testing.T) {
	t.Parallel()

	dir := writePolicy(t, "require-name.rego", allowIfNamed)
	v, err := newOPAValidator(context.Background(), dir, "require-name.rego")
	require.NoError(t, err)

	ok, msgs, warnings, err := v.ValidateEvaluation(context.Background(), &policyv1.RecordState{Record: mustStruct(t, map[string]any{"name": "agent"})})
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
	require.Empty(t, warnings)

	ok, msgs, _, err = v.ValidateEvaluation(context.Background(), &policyv1.RecordState{Record: mustStruct(t, map[string]any{"name": ""})})
	require.NoError(t, err)
	require.False(t, ok)
	require.Contains(t, msgs[0], "record rejected by policy")
}

func TestOPAValidator_ReadsRecordState(t *testing.T) {
	t.Parallel()

	const src = `
package dir.policy.require_state

default allow := false

allow if {
	input.record.name == "agent"
	input.record_cid == "bafy"
	input.meta.signature.status == "SIGNATURE_STATUS_VERIFIED"
}
`

	dir := writePolicy(t, "require-state.rego", src)
	v, err := newOPAValidator(context.Background(), dir, "require-state.rego")
	require.NoError(t, err)

	state := &policyv1.RecordState{
		Record:    mustStruct(t, map[string]any{"name": "agent"}),
		RecordCid: "bafy",
		Meta: &policyv1.RecordMeta{
			Signature: &policyv1.SignatureFacts{
				Status: policyv1.SignatureStatus_SIGNATURE_STATUS_VERIFIED,
			},
		},
	}

	ok, msgs, _, err := v.ValidateEvaluation(context.Background(), state)
	require.NoError(t, err)
	require.True(t, ok, msgs)
}

func TestNewOPAValidator_RequiresDirAndFile(t *testing.T) {
	t.Parallel()

	_, err := newOPAValidator(context.Background(), "", "policy.rego")
	require.Error(t, err)
	require.Contains(t, err.Error(), "policy directory is required")

	_, err = newOPAValidator(context.Background(), t.TempDir(), "../x.rego")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be a filename")
}

func TestNewRegistry_OPA(t *testing.T) {
	t.Parallel()

	// An operation policy reads the record as the Rego input document.
	const src = `
package dir.policy.require_name

default allow := false

allow if {
	input.name != ""
}
`

	dir := writePolicy(t, "policy.rego", src)
	reg, err := NewRegistry(context.Background(), nil, policyconfig.Operations{{
		Provider: policyconfig.ProviderOPA,
		Ops:      []string{policyconfig.OpPush},
		File:     "policy.rego",
	}}, dir)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "test"})
	ok, msgs, err := reg.Decide(context.Background(), policyconfig.OpPush, record)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
}

func mustStruct(t *testing.T, fields map[string]any) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(fields)
	require.NoError(t, err)

	return s
}
