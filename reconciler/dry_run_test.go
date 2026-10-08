// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agntcy/dir/reconciler/dryrun"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const candidatePolicy = `
policy:
  validators:
    - provider: cel
      op: ["evaluate"]
      config:
        name: has-description
        expressions:
          - 'record.description != ""'
`

// writeCandidate writes a valid candidate policy file and returns its path.
func writeCandidate(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "candidate.yaml")
	require.NoError(t, os.WriteFile(path, []byte(candidatePolicy), 0o600))

	return path
}

// useNode points the reconciler's configuration at a database in a temporary
// directory and at a registry at address, the way the node's environment would.
func useNode(t *testing.T, address string) {
	t.Helper()

	t.Setenv("RECONCILER_DATABASE_TYPE", "sqlite")
	t.Setenv("RECONCILER_DATABASE_SQLITE_PATH", filepath.Join(t.TempDir(), "dir.db"))
	t.Setenv("RECONCILER_POLICY_DIR", t.TempDir())
	t.Setenv("RECONCILER_LOCAL_REGISTRY_REGISTRY_ADDRESS", address)
	t.Setenv("RECONCILER_LOCAL_REGISTRY_REPOSITORY_NAME", "dir")
	t.Setenv("RECONCILER_LOCAL_REGISTRY_AUTH_CONFIG_INSECURE", "true")
}

// captureStdout redirects os.Stdout, where the report is written, and returns
// a function that restores it and returns what was written.
func captureStdout(t *testing.T) func() string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	original := os.Stdout
	os.Stdout = writer

	return func() string {
		os.Stdout = original

		require.NoError(t, writer.Close())

		out, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())

		return string(out)
	}
}

// closedAddress returns a local address nothing listens on.
func closedAddress(t *testing.T) string {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()

	require.NoError(t, listener.Close())

	return address
}

func TestRunDryRunRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		args func(t *testing.T) []string
		want string
	}{
		{
			name: "an unknown flag",
			args: func(*testing.T) []string { return []string{"--no-such-flag"} },
			want: "parse flags",
		},
		{
			name: "no candidate",
			args: func(*testing.T) []string { return nil },
			want: "a candidate file is required",
		},
		{
			name: "a candidate that cannot be read",
			args: func(t *testing.T) []string {
				t.Helper()

				return []string{"--candidate", filepath.Join(t.TempDir(), "missing.yaml")}
			},
			want: "missing.yaml",
		},
		{
			name: "an unknown output format",
			args: func(t *testing.T) []string {
				t.Helper()

				return []string{"--candidate", writeCandidate(t), "--output", "xml"}
			},
			want: `unknown output "xml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The candidate is checked before the node is opened, so no node is
			// needed: an address nothing answers at proves it is not reached.
			useNode(t, closedAddress(t))

			err := runDryRun(tt.args(t))

			require.ErrorContains(t, err, tt.want)
			assert.NotErrorIs(t, err, dryrun.ErrStoreUnreachable, "the node must not be opened for a bad candidate")
		})
	}
}

func TestRunDryRunNeedsARunningNode(t *testing.T) {
	useNode(t, closedAddress(t))

	err := runDryRun([]string{"--candidate", writeCandidate(t)})

	require.ErrorIs(t, err, dryrun.ErrStoreUnreachable)
	require.ErrorContains(t, err, "is the node running?")
}

func TestRunDryRunReportsOnAnEmptyNode(t *testing.T) {
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(http.StatusOK)

			return
		}

		http.NotFound(w, r)
	}))
	t.Cleanup(registry.Close)

	useNode(t, strings.TrimPrefix(registry.URL, "http://"))

	read := captureStdout(t)
	err := runDryRun([]string{"--candidate", writeCandidate(t), "--output", "json"})
	out := read()

	require.NoError(t, err)
	assert.Contains(t, out, `"policy_id": "cel:has-description"`)
	assert.Contains(t, out, `"evaluated": 0`)
}
