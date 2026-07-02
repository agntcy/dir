// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleDoc = `---
name: agntcy-dir
description: Use this skill to interact with DIR.
metadata:
  author: AGNTCY Contributors
  version: 1.0.0
---

# AGNTCY Directory (DIR)

Body content here.
`

func TestSplitFrontmatterExtractsFields(t *testing.T) {
	fm, body, err := splitFrontmatter(sampleDoc)
	require.NoError(t, err)

	assert.Equal(t, "agntcy-dir", fm.Name)
	assert.Equal(t, "Use this skill to interact with DIR.", fm.Description)
	assert.Contains(t, body, "# AGNTCY Directory (DIR)")
	assert.NotContains(t, body, "name: agntcy-dir")
}

func TestSplitFrontmatterNoFrontmatterReturnsWholeBody(t *testing.T) {
	doc := "# Title\n\nNo frontmatter.\n"

	fm, body, err := splitFrontmatter(doc)
	require.NoError(t, err)

	assert.Empty(t, fm.Name)
	assert.Equal(t, doc, body)
}
