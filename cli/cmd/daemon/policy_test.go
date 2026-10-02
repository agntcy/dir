// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/agntcy/dir/reconciler/dryrun"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicyDryRunIsUnderDaemonPolicy(t *testing.T) {
	cmd, rest, err := Command.Find([]string{"policy", "dry-run"})

	require.NoError(t, err)
	require.Empty(t, rest)
	assert.Equal(t, "dry-run", cmd.Name())

	for _, flag := range []string{"candidate", "policy-dir", "policy", "samples", "output"} {
		assert.NotNil(t, cmd.Flags().Lookup(flag), "--%s", flag)
	}

	// The candidate is what the command is for.
	required := cmd.Flags().Lookup("candidate").Annotations["cobra_annotation_bash_completion_one_required_flag"]
	assert.Equal(t, []string{"true"}, required)
}

// The records are read through the daemon's registry, so a dry run with no
// daemon to answer is refused, and says why, rather than reporting every record
// as one the policy could not judge.
func TestPolicyDryRunNeedsTheDaemonRunning(t *testing.T) {
	originalOpts, originalDryRun := opts, dryRunOpts

	t.Cleanup(func() {
		opts, dryRunOpts = originalOpts, originalDryRun
	})

	dir := t.TempDir()
	candidate := filepath.Join(dir, "candidate.yaml")
	config := filepath.Join(dir, "daemon.yaml")

	require.NoError(t, os.WriteFile(candidate, []byte(`
validators:
  - provider: cel
    op: ["evaluate"]
    config:
      name: has-description
      expressions:
        - 'record.description != ""'
`), 0o600))

	// A registry nothing listens at.
	require.NoError(t, os.WriteFile(config, []byte(`
server:
  store:
    provider: "oci"
    oci:
      local_dir: "store"
      registry_address: "127.0.0.1:1"
      repository_name: "dir"
      auth_config:
        insecure: true
  database:
    type: "sqlite"
    sqlite:
      path: "dir.db"
`), 0o600))

	opts = &Options{DataDir: dir, ConfigFile: config}
	dryRunOpts = dryrun.Options{Candidate: candidate, Samples: 5, Output: dryrun.OutputHuman}

	policyDryRunCmd.SetOut(&bytes.Buffer{})
	policyDryRunCmd.SetContext(t.Context())

	err := runPolicyDryRun(policyDryRunCmd, nil)

	require.ErrorIs(t, err, dryrun.ErrStoreUnreachable)
	require.ErrorContains(t, err, "127.0.0.1:1", "it says where it looked")
	require.ErrorContains(t, err, "is the node running?")
}

func TestPolicyDryRunRefusesAFileWithNoPolicy(t *testing.T) {
	originalOpts, originalDryRun := opts, dryRunOpts

	t.Cleanup(func() {
		opts, dryRunOpts = originalOpts, originalDryRun
	})

	candidate := filepath.Join(t.TempDir(), "none.yaml")
	require.NoError(t, os.WriteFile(candidate, []byte("validators: []\n"), 0o600))

	opts = &Options{DataDir: t.TempDir()}
	dryRunOpts = dryrun.Options{Candidate: candidate, Samples: 5, Output: dryrun.OutputHuman}

	policyDryRunCmd.SetOut(&bytes.Buffer{})
	policyDryRunCmd.SetContext(t.Context())

	err := runPolicyDryRun(policyDryRunCmd, nil)

	require.ErrorContains(t, err, "defines no policy")
}
