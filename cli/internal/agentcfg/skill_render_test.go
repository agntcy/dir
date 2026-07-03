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
	// The name is YAML-quoted so names with YAML-significant characters stay valid.
	assert.Contains(t, s, `name: "agntcy-dir"`)
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

// --- New tests ---

func TestRenderClineReturnsBodyOnly(t *testing.T) {
	out, err := renderCline(sampleDoc)
	require.NoError(t, err)

	s := string(out)
	// No frontmatter delimiters.
	assert.False(t, strings.HasPrefix(s, "---"))
	assert.NotContains(t, s, "name: agntcy-dir")
	// Body content is present.
	assert.Contains(t, s, "# AGNTCY Directory (DIR)")
	assert.Contains(t, s, "Body content here.")
}

func TestSplitFrontmatterCRLF(t *testing.T) {
	// Build a CRLF version of sampleDoc.
	crlf := strings.ReplaceAll(sampleDoc, "\n", "\r\n")
	fm, body, err := splitFrontmatter(crlf)
	require.NoError(t, err)

	assert.Equal(t, "agntcy-dir", fm.Name)
	assert.Equal(t, "Use this skill to interact with DIR.", fm.Description)
	assert.Contains(t, body, "# AGNTCY Directory (DIR)")
}

func TestSplitFrontmatterMalformedYAML(t *testing.T) {
	// Malformed YAML in the frontmatter block.
	doc := "---\nkey: [unclosed\n---\n\nbody\n"
	_, _, err := splitFrontmatter(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse skill frontmatter")
}

func TestRenderContinueQuotesSpecialCharsInName(t *testing.T) {
	// A name containing "a: b" is YAML-significant; strconv.Quote must escape it.
	doc := "---\nname: \"a: b\"\ndescription: desc\n---\n\nbody\n"
	out, err := renderContinue(doc)
	require.NoError(t, err)

	s := string(out)
	// The name should be properly quoted so the YAML is still valid.
	assert.Contains(t, s, `name: "a: b"`)
	assert.Contains(t, s, "alwaysApply: true")
}
