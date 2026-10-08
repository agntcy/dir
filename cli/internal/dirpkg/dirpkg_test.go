// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package dirpkg_test

import (
	"testing"

	"github.com/agntcy/dir/cli/internal/dirpkg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactsCarriesTheSkillOnly(t *testing.T) {
	t.Parallel()

	arts, err := dirpkg.Artifacts()
	require.NoError(t, err)

	assert.NotEmpty(t, dirpkg.Name())
	assert.NotEmpty(t, dirpkg.Version())
	assert.Empty(t, arts.MCPServerNames(), "the built-in package no longer installs an MCP server")
	assert.True(t, arts.HasSkill())
}
