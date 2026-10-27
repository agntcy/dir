// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package importcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDefaultConfigWritesOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", DefaultConfigFile)

	require.NoError(t, ensureDefaultConfig(path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.YAMLEq(t, defaultConfigYAML, string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(configFilePerm), info.Mode().Perm())

	// An edited file must never be overwritten.
	require.NoError(t, os.WriteFile(path, []byte("limit: 7\n"), configFilePerm))
	require.NoError(t, ensureDefaultConfig(path))

	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "limit: 7\n", string(data))
}

// TestDefaultConfigUsesLLMEnricher ensures the written default is the reference
// configuration with the LLM enricher.
func TestDefaultConfigUsesLLMEnricher(t *testing.T) {
	path := filepath.Join(t.TempDir(), DefaultConfigFile)
	require.NoError(t, ensureDefaultConfig(path))

	o, flags := newTestOptions(t)
	o.implicitConfigFile = path
	require.NoError(t, o.loadConfig(flags))

	require.NotNil(t, o.Enricher.LLM)
	assert.Nil(t, o.Enricher.Static)
	require.NoError(t, o.Enricher.Validate())
	assert.Equal(t, "1.1.0", o.SchemaVersion)
	assert.Equal(t, defaultToolHostConfig(), o.Enricher.LLM.ToolHost)
	assert.Equal(t, 5, o.Enricher.LLM.RequestsPerMinute)
	assert.Equal(t, "mcp-registry", string(o.Type))
}

func TestExplicitConfigOverridesImplicit(t *testing.T) {
	implicit := filepath.Join(t.TempDir(), DefaultConfigFile)
	require.NoError(t, ensureDefaultConfig(implicit))

	explicit := writeConfig(t, "limit: 5\n")

	o, flags := newTestOptions(t, "--config", explicit)
	o.implicitConfigFile = implicit
	require.NoError(t, o.loadConfig(flags))

	assert.Equal(t, 5, o.Limit)
	assert.Empty(t, o.SchemaVersion)
	// The implicit file is ignored entirely when --config is given.
	require.NotNil(t, o.Enricher.LLM)
}
