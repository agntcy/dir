// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSelectionAllIsEmptyMap(t *testing.T) {
	chosen, err := ParseSelection([]string{AllAgents})
	require.NoError(t, err)
	assert.Empty(t, chosen)
}

func TestParseSelectionExplicitList(t *testing.T) {
	chosen, err := ParseSelection([]string{"claude-code", " cursor "})
	require.NoError(t, err)
	assert.True(t, chosen["claude-code"])
	assert.True(t, chosen["cursor"])
	assert.Len(t, chosen, 2)
}

func TestParseSelectionUnknownID(t *testing.T) {
	_, err := ParseSelection([]string{"nope"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown agent")
}

func TestParseSelectionAllWithIDConflicts(t *testing.T) {
	_, err := ParseSelection([]string{AllAgents, "cursor"})
	require.Error(t, err)
}

func TestAgentIDsMatchRegistry(t *testing.T) {
	assert.Len(t, AgentIDs(), len(Registry()))
}
