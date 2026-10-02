// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dryrun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/reconciler/tasks/policy"
	"github.com/agntcy/dir/server/database"
	dbconfig "github.com/agntcy/dir/server/database/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	servertypes "github.com/agntcy/dir/server/types"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const candidateCEL = `
validators:
  - provider: cel
    op: ["evaluate"]
    config:
      name: has-description
      expressions:
        - 'record.description != ""'
`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

// store serves the records it holds.
type store struct{ records map[string]*corev1.Record }

func (s store) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	r, ok := s.records[ref.GetCid()]
	if !ok {
		return nil, servertypes.RecordNotFoundError(ref.GetCid()) //nolint:wrapcheck // the store's own status
	}

	return r, nil
}

// node is a database holding three records, one without a description, and the
// store they are in.
type node struct {
	db      servertypes.DatabaseAPI
	records store
	bare    string // the CID of the record without a description
}

func newNode(t *testing.T) node {
	t.Helper()

	db, err := database.New(dbconfig.Config{Type: "sqlite", SQLite: dbconfig.SQLiteConfig{Path: filepath.Join(t.TempDir(), "dir.db")}})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	n := node{db: db, records: store{records: map[string]*corev1.Record{}}}

	for _, rec := range []struct{ name, description string }{
		{"described-a", "does something"},
		{"bare", ""},
		{"described-b", "does something else"},
	} {
		record := corev1.New(&typesv1alpha1.Record{Name: rec.name, Description: rec.description, SchemaVersion: "0.7.0"})

		adapter, err := record.Decode()
		require.NoError(t, err)
		require.NoError(t, db.AddRecord(adapter))

		n.records.records[record.GetCid()] = record

		if rec.name == "bare" {
			n.bare = record.GetCid()
		}
	}

	return n
}

func (n node) source() Source {
	return Source{DB: n.db, Records: n.records}
}

func TestRun_ReportsWhatTheCandidateWouldExclude(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", candidateCEL)

	var out bytes.Buffer

	err := Run(t.Context(), Options{Candidate: candidate, Samples: 10, Output: OutputHuman}, n.source(), &out)

	require.NoError(t, err)

	text := out.String()
	assert.Contains(t, text, "Policy cel:has-description, version ")
	assert.Contains(t, text, "Evaluated:      3 records")
	assert.Contains(t, text, "Would pass:     2")
	assert.Contains(t, text, "Would exclude:  1 (1 rejected by the policy, 0 it could not evaluate)")
	assert.Contains(t, text, n.bare, "the record it would exclude is listed")
	assert.Contains(t, text, "record.description", "with the reason the policy gave")
	assert.Contains(t, text, "Nothing was stored")
}

// The acceptance of the dry run: it changes no stored verdict, and registers no
// version, so what a read returns cannot change.
func TestRun_ChangesNothingTheNodeHolds(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", candidateCEL)

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, Samples: 10, Output: OutputHuman}, n.source(), &bytes.Buffer{}))

	// Under the policy's own ID no verdict was stored: every record still needs one.
	needing, err := n.db.GetRecordsNeedingPolicyEvaluation("cel:has-description", "any-version", "", 100)
	require.NoError(t, err)
	assert.Len(t, needing, 3, "no verdict was stored")

	needing, err = n.db.GetRecordsNeedingPolicyEvaluation("dry-run:cel:has-description", "any-version", "", 100)
	require.NoError(t, err)
	assert.Len(t, needing, 3, "nor under the ID the dry run selects with")

	_, registered, err := n.db.GetCurrentPolicyVersion("cel:has-description")
	require.NoError(t, err)
	assert.False(t, registered, "no policy version was registered, so no server follows it")
}

func TestRun_JSON(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", candidateCEL)

	var out bytes.Buffer

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, Samples: 10, Output: OutputJSON}, n.source(), &out))

	var got struct {
		Policies []struct {
			PolicyID     string `json:"policy_id"`
			Evaluated    int    `json:"evaluated"`
			Compliant    int    `json:"compliant"`
			NonCompliant int    `json:"non_compliant"`
			WouldExclude int    `json:"would_exclude"`
			Sample       []struct {
				CID    string `json:"cid"`
				Reason string `json:"reason"`
			} `json:"sample"`
		} `json:"policies"`
	}

	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	require.Len(t, got.Policies, 1)

	p := got.Policies[0]
	assert.Equal(t, "cel:has-description", p.PolicyID)
	assert.Equal(t, 3, p.Evaluated)
	assert.Equal(t, 2, p.Compliant)
	assert.Equal(t, 1, p.NonCompliant)
	assert.Equal(t, 1, p.WouldExclude)
	require.Len(t, p.Sample, 1)
	assert.Equal(t, n.bare, p.Sample[0].CID)
}

func TestRun_ListsOnlyTheSamplesAskedFor(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", `
validators:
  - provider: cel
    op: ["evaluate"]
    config:
      name: nothing-passes
      expressions: ['false']
`)

	var out bytes.Buffer

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, Samples: 1, Output: OutputHuman}, n.source(), &out))

	assert.Contains(t, out.String(), "Would exclude:  3", "the count is of every record")
	assert.Contains(t, out.String(), "first 1 of 3; --samples lists more", "and the list says it is part of them")
}

func TestRun_OPAPolicyFromADirectory(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	dir := t.TempDir()
	writeFile(t, dir, "require-description.rego", `
package dir.policy.require_description

default allow := false

allow if {
	input.description != ""
}
`)
	candidate := writeFile(t, dir, "candidate.yaml", `
validators:
  - provider: opa
    op: ["evaluate"]
    config:
      file: require-description.rego
`)

	var out bytes.Buffer

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, PolicyDir: dir, Samples: 10, Output: OutputHuman}, n.source(), &out))

	assert.Contains(t, out.String(), "Policy opa:require-description, version ")
	assert.Contains(t, out.String(), "Would exclude:  1")
}

func TestRun_TriesOnlyThePolicyNamed(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", `
validators:
  - provider: cel
    op: ["evaluate"]
    config: {name: a, expressions: ['true']}
  - provider: cel
    op: ["evaluate"]
    config: {name: b, expressions: ['false']}
`)

	var out bytes.Buffer

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, Policy: "cel:b", Samples: 0, Output: OutputHuman}, n.source(), &out))

	assert.Contains(t, out.String(), "Policy cel:b,")
	assert.NotContains(t, out.String(), "Policy cel:a,")

	err := Run(t.Context(), Options{Candidate: candidate, Policy: "cel:c", Output: OutputHuman}, n.source(), &out)
	require.ErrorIs(t, err, errNoPolicyNamed)
	require.ErrorContains(t, err, "cel:a, cel:b", "it says what the file does define")
}

func TestRun_BothPoliciesOfAFileAreReported(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", `
validators:
  - provider: cel
    op: ["evaluate"]
    config: {name: a, expressions: ['true']}
  - provider: cel
    op: ["evaluate"]
    config: {name: b, expressions: ['false']}
`)

	var out bytes.Buffer

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, Output: OutputHuman}, n.source(), &out))

	assert.Contains(t, out.String(), "Policy cel:a,")
	assert.Contains(t, out.String(), "Policy cel:b,")
}

func TestRun_RejectsBadOptions(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", candidateCEL)

	for name, opts := range map[string]Options{
		"no candidate":     {Output: OutputHuman},
		"negative samples": {Candidate: candidate, Samples: -1, Output: OutputHuman},
		"unknown output":   {Candidate: candidate, Output: "yaml"},
	} {
		err := Run(t.Context(), opts, n.source(), &bytes.Buffer{})

		require.Error(t, err, name)
	}
}

func TestLoadCandidate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	t.Run("keeps only the entries that define a policy", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, dir, "mixed.yaml", `
validators:
  - provider: oasf
    op: ["push"]
    config: {schema_url: "https://schema.example.com"}
  - provider: cel
    op: ["push", "evaluate"]
    config: {name: p, expressions: ['true']}
`)

		got, err := LoadCandidate(path)

		require.NoError(t, err)
		require.Len(t, got, 1)
		assert.Equal(t, "cel:p", got[0].PolicyID())
	})

	t.Run("a file with no policy is refused, and says what is missing", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, dir, "none.yaml", `
validators:
  - provider: cel
    op: ["push"]
    config: {expressions: ['true']}
`)

		_, err := LoadCandidate(path)

		require.ErrorContains(t, err, `op: ["evaluate"]`)
	})

	t.Run("an invalid policy is refused", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, dir, "invalid.yaml", `
validators:
  - provider: cel
    op: ["evaluate"]
    config: {expressions: ['true']}
`)

		_, err := LoadCandidate(path)

		require.ErrorContains(t, err, "config.name is required")
	})

	t.Run("two policies with one ID are refused", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, dir, "dup.yaml", `
validators:
  - provider: cel
    op: ["evaluate"]
    config: {name: p, expressions: ['true']}
  - provider: cel
    op: ["evaluate"]
    config: {name: p, expressions: ['false']}
`)

		_, err := LoadCandidate(path)

		require.ErrorContains(t, err, "already defined")
	})

	t.Run("a file with no extension is read as YAML, as standard input is", func(t *testing.T) {
		t.Parallel()

		path := writeFile(t, dir, "candidate", candidateCEL)

		got, err := LoadCandidate(path)

		require.NoError(t, err)
		require.Len(t, got, 1)
	})

	t.Run("a file that is not there", func(t *testing.T) {
		t.Parallel()

		_, err := LoadCandidate(filepath.Join(dir, "missing.yaml"))

		require.ErrorContains(t, err, "read candidate file")
	})
}

func TestBindFlags(t *testing.T) {
	t.Parallel()

	var opts Options

	fs := pflag.NewFlagSet("dry-run", pflag.ContinueOnError)
	BindFlags(fs, &opts)

	require.NoError(t, fs.Parse(nil))
	assert.Empty(t, opts.PolicyDir, "left for the caller to fill with the node's policy directory")
	assert.Equal(t, DefaultSamples, opts.Samples)
	assert.Equal(t, OutputHuman, opts.Output)

	require.NoError(t, fs.Parse([]string{"--candidate", "c.yaml", "--policy-dir", "/tmp/p", "--policy", "opa:x", "--samples", "3", "--output", "json"}))
	assert.Equal(t, Options{Candidate: "c.yaml", PolicyDir: "/tmp/p", Policy: "opa:x", Samples: 3, Output: OutputJSON}, opts)
}

// A store nothing answers is refused up front: a record that cannot be read is
// not a record the policy rejects.
func TestOpen_RefusesAStoreNothingAnswers(t *testing.T) {
	t.Parallel()

	// A registry nothing listens at, reached without TLS.
	unreachable := ociconfig.Config{RegistryAddress: "127.0.0.1:1", RepositoryName: "dir", Insecure: true}

	_, _, err := Open(t.Context(),
		dbconfig.Config{Type: "sqlite", SQLite: dbconfig.SQLiteConfig{Path: filepath.Join(t.TempDir(), "dir.db")}},
		unreachable,
		policy.Config{})

	require.ErrorIs(t, err, ErrStoreUnreachable)
	require.ErrorContains(t, err, "is the node running?")
}

// The candidate is checked without a node, so that a mistake in it is found
// before anything is opened.
func TestPrepare_NeedsNoNode(t *testing.T) {
	t.Parallel()

	good := writeFile(t, t.TempDir(), "good.yaml", candidateCEL)

	prepared, err := Prepare(t.Context(), Options{Candidate: good, Samples: 5, Output: OutputHuman})
	require.NoError(t, err)
	assert.NotNil(t, prepared)

	bad := writeFile(t, t.TempDir(), "bad.yaml", "validators: []\n")

	_, err = Prepare(t.Context(), Options{Candidate: bad, Output: OutputHuman})
	require.ErrorContains(t, err, "defines no policy")
}

// Records the policy could not judge may mean the store could not be read; the
// report says so, rather than leaving it to be guessed.
func TestRun_HintsThatUnreadableRecordsAreNotThePolicysFault(t *testing.T) {
	t.Parallel()

	n := newNode(t)
	n.records = store{records: map[string]*corev1.Record{}} // the store holds none of them
	candidate := writeFile(t, t.TempDir(), "candidate.yaml", candidateCEL)

	var out bytes.Buffer

	require.NoError(t, Run(t.Context(), Options{Candidate: candidate, Samples: 2, Output: OutputHuman}, n.source(), &out))

	assert.Contains(t, out.String(), "Would exclude:  3 (0 rejected by the policy, 3 it could not evaluate)")
	assert.Contains(t, out.String(), "may mean the store could not be\n  read")
	assert.Contains(t, out.String(), "could not be evaluated: read record")
}

// compile-time: the dry run is given nothing it could write through.
var _ policy.DryRunDB = servertypes.DatabaseAPI(nil)
