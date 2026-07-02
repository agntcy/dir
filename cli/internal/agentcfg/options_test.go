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

func selectedIDs(sels []Selection) []string {
	ids := make([]string, 0, len(sels))
	for _, s := range sels {
		ids = append(ids, s.Agent.ID)
	}

	return ids
}

func TestResolveSelectionDefaultUsesDetected(t *testing.T) {
	sels := ResolveSelection(fakeAgents(), Env{}, nil, false, false)
	assert.Equal(t, []string{"a", "c"}, selectedIDs(sels))

	for _, s := range sels {
		assert.False(t, s.Forced)
	}
}

func TestResolveSelectionExplicitFlagsForceChosen(t *testing.T) {
	chosen := map[string]bool{"b": true}

	sels := ResolveSelection(fakeAgents(), Env{}, chosen, false, false)
	assert.Equal(t, []string{"b"}, selectedIDs(sels))
	assert.True(t, sels[0].Forced)
}

func TestResolveSelectionAllUsesDetected(t *testing.T) {
	sels := ResolveSelection(fakeAgents(), Env{}, nil, true, false)
	assert.Equal(t, []string{"a", "c"}, selectedIDs(sels))
}

func TestResolveSelectionForceWithoutFlagsActsOnAll(t *testing.T) {
	sels := ResolveSelection(fakeAgents(), Env{}, nil, false, true)
	assert.Equal(t, []string{"a", "b", "c"}, selectedIDs(sels))

	for _, s := range sels {
		assert.True(t, s.Forced)
	}
}

func TestResolveArtifacts(t *testing.T) {
	assert.Equal(t, ArtifactSet{MCP: true, Skill: true}, ResolveArtifacts(false, false))
	assert.Equal(t, ArtifactSet{MCP: true, Skill: true}, ResolveArtifacts(true, true))
	assert.Equal(t, ArtifactSet{MCP: true, Skill: false}, ResolveArtifacts(true, false))
	assert.Equal(t, ArtifactSet{MCP: false, Skill: true}, ResolveArtifacts(false, true))
}
