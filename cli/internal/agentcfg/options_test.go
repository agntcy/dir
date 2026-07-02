// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func fakeAgents() []Agent {
	return []Agent{
		{ID: "a", Name: "A", Flag: "a", Detect: func(Env) bool { return true }},
		{ID: "b", Name: "B", Flag: "b", Detect: func(Env) bool { return false }},
		{ID: "c", Name: "C", Flag: "c", Detect: func(Env) bool { return true }},
	}
}

func agentIDs(agents []Agent) []string {
	ids := make([]string, 0, len(agents))
	for _, a := range agents {
		ids = append(ids, a.ID)
	}

	return ids
}

func TestResolveSelectionEmptyChosenUsesDetected(t *testing.T) {
	selected, skipped := ResolveSelection(fakeAgents(), Env{}, nil)
	assert.Equal(t, []string{"a", "c"}, agentIDs(selected))
	assert.Empty(t, skipped)
}

func TestResolveSelectionExplicitChosenOnlyDetected(t *testing.T) {
	// "a" is detected, "b" is not; both requested.
	chosen := map[string]bool{"a": true, "b": true}

	selected, skipped := ResolveSelection(fakeAgents(), Env{}, chosen)
	assert.Equal(t, []string{"a"}, agentIDs(selected))
	assert.Equal(t, []string{"b"}, skipped)
}

func TestResolveSelectionUndetectedNeverInstalled(t *testing.T) {
	// Requesting only an undetected agent selects nothing and reports it skipped.
	chosen := map[string]bool{"b": true}

	selected, skipped := ResolveSelection(fakeAgents(), Env{}, chosen)
	assert.Empty(t, selected)
	assert.Equal(t, []string{"b"}, skipped)
}

func TestResolveArtifacts(t *testing.T) {
	assert.Equal(t, ArtifactSet{MCP: true, Skill: true}, ResolveArtifacts(false, false))
	assert.Equal(t, ArtifactSet{MCP: true, Skill: true}, ResolveArtifacts(true, true))
	assert.Equal(t, ArtifactSet{MCP: true, Skill: false}, ResolveArtifacts(true, false))
	assert.Equal(t, ArtifactSet{MCP: false, Skill: true}, ResolveArtifacts(false, true))
}
