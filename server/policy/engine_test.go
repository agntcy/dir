// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"os"
	"path/filepath"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/policy/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const requireOwnerRego = `# METADATA
# scope: package
# title: require-owner
# custom:
#   triggers: ["ingest"]
package dir.policy.require_owner

default allow := false

allow if {
	input.record.annotations["org.agntcy/owner"]
}
`

const publishOnlyRego = `# METADATA
# scope: package
# title: publish-only
# custom:
#   triggers: ["publish"]
package dir.policy.publish_only

default allow := false
`

const noMetadataRego = `package dir.policy.no_meta

default allow := true
`

func TestLoadDisabled(t *testing.T) {
	t.Parallel()

	eval, err := Load(t.Context(), config.Config{})
	require.NoError(t, err)
	require.NoError(t, eval.EvaluateIngest(t.Context(), "push", newRecord(nil)))
}

func TestLoadMissingDir(t *testing.T) {
	t.Parallel()

	_, err := Load(t.Context(), config.Config{Enabled: true, Dir: filepath.Join(t.TempDir(), "missing")})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read policy directory")
}

func TestEvaluateIngest_AllowAndDeny(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "require-owner.rego"), []byte(requireOwnerRego), 0o600))

	eval, err := Load(t.Context(), config.Config{Enabled: true, Dir: dir})
	require.NoError(t, err)

	err = eval.EvaluateIngest(t.Context(), "push", newRecord(map[string]string{"org.agntcy/owner": "team"}))
	require.NoError(t, err)

	err = eval.EvaluateIngest(t.Context(), "push", newRecord(nil))
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), `record rejected by policy "require-owner"`)
}

func TestLoadSkipsNonIngestTriggers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "publish-only.rego"), []byte(publishOnlyRego), 0o600))

	eval, err := Load(t.Context(), config.Config{Enabled: true, Dir: dir})
	require.NoError(t, err)

	// No ingest policies compiled, so a record without annotations is admitted.
	require.NoError(t, eval.EvaluateIngest(t.Context(), "push", newRecord(nil)))
}

func TestLoadParseError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.rego"), []byte("not rego"), 0o600))

	_, err := Load(t.Context(), config.Config{Enabled: true, Dir: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse policy file")
}

func TestLoadRequiresTriggers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "no-meta.rego"), []byte(noMetadataRego), 0o600))

	_, err := Load(t.Context(), config.Config{Enabled: true, Dir: dir})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom.triggers is required")
}

func TestEvaluateIngest_NilRecordFailsClosed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "require-owner.rego"), []byte(requireOwnerRego), 0o600))

	eval, err := Load(t.Context(), config.Config{Enabled: true, Dir: dir})
	require.NoError(t, err)

	err = eval.EvaluateIngest(t.Context(), "push", nil)
	require.Error(t, err)
	assert.Equal(t, codes.FailedPrecondition, status.Code(err))
	assert.Contains(t, err.Error(), "content policy evaluation failed")
}

func newRecord(annotations map[string]string) *corev1.Record {
	return corev1.New(&typesv1alpha1.Record{
		Name:          "test-record",
		SchemaVersion: "0.7.0",
		Annotations:   annotations,
	})
}
