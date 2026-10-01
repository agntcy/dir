// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validators

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	validatorsconfig "github.com/agntcy/dir/server/validators/config"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

const allowIfNamed = `
package dir.policy.require_name

default allow := false

allow if {
	input.name != ""
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

	ok, msgs, warnings, err := v.ValidateRecord(context.Background(), mustStruct(t, map[string]any{"name": "agent"}))
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, msgs)
	require.Empty(t, warnings)

	ok, msgs, _, err = v.ValidateRecord(context.Background(), mustStruct(t, map[string]any{"name": ""}))
	require.NoError(t, err)
	require.False(t, ok)
	require.Contains(t, msgs[0], "record rejected by policy")
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

	dir := writePolicy(t, "policy.rego", allowIfNamed)
	reg, err := NewRegistry(context.Background(), validatorsconfig.Config{{
		Provider: validatorsconfig.ProviderOPA,
		Ops:      []string{validatorsconfig.OpPush},
		Config:   map[string]any{validatorsconfig.ConfigKeyFile: "policy.rego"},
	}}, dir)
	require.NoError(t, err)

	record := corev1.New(&typesv1alpha1.Record{Name: "test"})
	ok, msgs, err := reg.Run(context.Background(), validatorsconfig.OpPush, record)
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
