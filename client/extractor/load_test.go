// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"os"
	"path/filepath"
	"testing"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUnprovisionedErrors(t *testing.T) {
	// A fresh dir has no manifest, so Load must refuse without provisioning.
	_, err := Load(Config{OASFURL: "https://schema.oasf.outshift.com", AssetDir: t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not provisioned")
	assert.Contains(t, err.Error(), "dirctl init")
}

func TestLoadInvalidConfigErrors(t *testing.T) {
	_, err := Load(Config{OASFURL: "ftp://nope", AssetDir: t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "OASF URL")
}

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
	require.NoError(t, clientconfig.SaveExtractor("", &clientconfig.Extractor{
		OASFURL:  "https://schema.oasf.outshift.com",
		AssetDir: assetDir,
	}))

	_, err := LoadConfigured()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not provisioned")
}
