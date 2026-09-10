// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"fmt"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestApplyTargetsKeepsOutcomesPerRecord(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	opts.pin = false

	targets := []installTarget{
		{label: "a:1.0.0", record: corev1.New(&oasfv1alpha1.Record{Name: "a", Version: "1.0.0"})},
		{label: "b:1.0.0", record: corev1.New(&oasfv1alpha1.Record{Name: "b", Version: "1.0.0"})},
	}

	// A stub apply: one outcome per call, so each record's outcomes are
	// distinguishable in the result.
	calls := 0
	apply := func(_ agentcfg.Env, _ agentinstall.Artifacts, _ []agentcfg.Agent, _ agentcfg.Scope, _ bool) []agentcfg.Outcome {
		calls++

		return []agentcfg.Outcome{{
			Agent:    "Claude Code",
			Artifact: agentcfg.ArtifactSkill,
			Path:     fmt.Sprintf("/skills/%d", calls),
			Action:   agentcfg.ActionAdded,
		}}
	}

	items, outcomes := applyTargets(agentcfg.Env{}, targets, nil, agentcfg.Global, false, apply)

	// Per record, so a manifest row is built from that record's own outcomes.
	require.Len(t, items, 2)
	require.Len(t, items[0].outcomes, 1)
	assert.Equal(t, "a", items[0].record.GetName())
	assert.Equal(t, "/skills/1", items[0].outcomes[0].Path)
	assert.Equal(t, "/skills/2", items[1].outcomes[0].Path)

	// Flattened and label-tagged, which is what the plan and summary group on.
	require.Len(t, outcomes, 2)
	assert.Equal(t, "a:1.0.0", outcomes[0].Record)
	assert.Equal(t, "b:1.0.0", outcomes[1].Record)
}

func TestApplyTargetsPinsOnlyWhenAsked(t *testing.T) {
	orig := opts

	defer func() { opts = orig }()

	targets := []installTarget{{label: "a", record: corev1.New(&oasfv1alpha1.Record{Name: "a"})}}
	apply := func(_ agentcfg.Env, _ agentinstall.Artifacts, _ []agentcfg.Agent, _ agentcfg.Scope, _ bool) []agentcfg.Outcome {
		return nil
	}

	// Batch mode has no explicit :version to imply a pin.
	opts.pin = false
	items, _ := applyTargets(agentcfg.Env{}, targets, nil, agentcfg.Global, false, apply)
	require.Len(t, items, 1)
	assert.False(t, items[0].pinned)

	opts.pin = true
	items, _ = applyTargets(agentcfg.Env{}, targets, nil, agentcfg.Global, false, apply)
	require.Len(t, items, 1)
	assert.True(t, items[0].pinned)
}

func TestUnsuitableRecordsAreRejectedBeforeInstall(t *testing.T) {
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
