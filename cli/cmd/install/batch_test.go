// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/stretchr/testify/require"
)

func TestResolveBatchOrInputMutuallyExclusive(t *testing.T) {
	err := resolveBatchOrInput(true, true, nil, nil, nil)
	require.ErrorIs(t, err, errBatchInputConflict)
}

func TestResolveBatchOrInputBatch(t *testing.T) {
	called := false

	err := resolveBatchOrInput(false, true, func() error {
		called = true

		return nil
	}, nil, nil)
	require.NoError(t, err)
	require.True(t, called)
}

func TestResolveBatchOrInputSingle(t *testing.T) {
	called := false

	err := resolveBatchOrInput(true, false, nil, func() error {
		called = true

		return nil
	}, nil)
	require.NoError(t, err)
	require.True(t, called)
}

func TestRecordLabel(t *testing.T) {
	require.Equal(t, "agent-a:1.0.0", getRecordLabel(corev1.New(&oasfv1alpha1.Record{
		Name:    "agent-a",
		Version: "1.0.0",
	})))
	require.Equal(t, "agent-b", getRecordLabel(corev1.New(&oasfv1alpha1.Record{Name: "agent-b"})))
}

func TestSelectRecordsLatestByName(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	opts.allVersions = false

	recs := []*corev1.Record{
		corev1.New(&oasfv1alpha1.Record{Name: "a", Version: "1.0.0"}),
		corev1.New(&oasfv1alpha1.Record{Name: "a", Version: "2.0.0"}),
		corev1.New(&oasfv1alpha1.Record{Name: "b", Version: "1.0.0"}),
	}

	selected := selectRecords(recs)
	require.Len(t, selected, 2)
	require.Equal(t, "2.0.0", selected[0].GetVersion())
	require.Equal(t, "b", selected[1].GetName())
}

func TestSelectRecordsAllVersions(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	opts.allVersions = true

	recs := []*corev1.Record{
		corev1.New(&oasfv1alpha1.Record{Name: "a", Version: "1.0.0"}),
		corev1.New(&oasfv1alpha1.Record{Name: "a", Version: "2.0.0"}),
	}

	require.Len(t, selectRecords(recs), 2)
}

func TestFormatSkippedSummary(t *testing.T) {
	out := formatSkippedSummary([]skippedRecord{
		{label: "bare", reason: "no installable module"},
	})
	require.Contains(t, out, "Skipped records")
	require.Contains(t, out, "bare")
	require.Contains(t, out, "no installable module")
}

func TestBatchInstallSkipsUnsuitableRecords(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	// Bare record: no modules, so DeriveArtifacts must reject it. Built inline
	// rather than via loadRecord/testdata, which moved with the derive/apply
	// logic into the agentinstall package.
	bare := corev1.New(&oasfv1alpha1.Record{
		Name:        "bare",
		Version:     "1.0.0",
		Description: "A record with no modules",
	})

	_, err := agentinstall.DeriveArtifacts(bare)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no installable")
}
