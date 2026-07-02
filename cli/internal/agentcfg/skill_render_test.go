// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderCursorHasFrontmatterAndBody(t *testing.T) {
	out, err := renderCursor(sampleDoc)
	require.NoError(t, err)

	s := string(out)
	assert.True(t, strings.HasPrefix(s, "---\n"))
	assert.Contains(t, s, "alwaysApply: true")
	assert.Contains(t, s, "Use this skill to interact with DIR.")
	assert.Contains(t, s, "# AGNTCY Directory (DIR)")
	// Original frontmatter keys must be replaced, not duplicated.
	assert.NotContains(t, s, "name: agntcy-dir")
}

func TestRenderCopilotHasApplyTo(t *testing.T) {
	out, err := renderCopilot(sampleDoc)
	require.NoError(t, err)
	assert.Contains(t, string(out), `applyTo: "**"`)
}

func TestRenderContinueHasNameAndAlwaysApply(t *testing.T) {
	out, err := renderContinue(sampleDoc)
	require.NoError(t, err)

	s := string(out)
	assert.Contains(t, s, "name: agntcy-dir")
	assert.Contains(t, s, "alwaysApply: true")
}

func TestRenderRooStripsFrontmatter(t *testing.T) {
	out, err := renderRoo(sampleDoc)
	require.NoError(t, err)

	s := string(out)
	assert.False(t, strings.HasPrefix(s, "---"))
	assert.Contains(t, s, "# AGNTCY Directory (DIR)")
	assert.NotContains(t, s, "alwaysApply")
}

func TestRenderManagedInnerIsBodyOnly(t *testing.T) {
	out, err := renderManagedInner(sampleDoc)
	require.NoError(t, err)

	s := string(out)
	assert.Contains(t, s, "# AGNTCY Directory (DIR)")
	assert.NotContains(t, s, "name: agntcy-dir")
}
