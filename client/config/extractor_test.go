// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfiguredNotConfiguredErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// No persisted extractor section: consumers get an actionable error, not a
	// provisioning attempt.
	_, err := LoadConfigured()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
	assert.Contains(t, err.Error(), "dirctl init")
}

func TestLoadConfiguredUnprovisionedErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// Config points at a dir that was never provisioned (no manifest on disk).
	assetDir := filepath.Join(t.TempDir(), "assets")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, SaveExtractor("", &Extractor{
		OASFURL:  "https://schema.oasf.outshift.com",
		AssetDir: assetDir,
	}))

	_, err := LoadConfigured()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not provisioned")
}
