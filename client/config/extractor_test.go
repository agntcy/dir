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

func TestResolveConfiguredNotConfiguredErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// No persisted extractor section: consumers get an actionable error, not a
	// provisioning attempt.
	_, err := ResolveConfigured()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
	assert.Contains(t, err.Error(), "dirctl init")
}

func TestResolveConfiguredUnprovisionedLocalErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// Local config (no RemoteAddr) pointing at a dir that was never provisioned.
	assetDir := filepath.Join(t.TempDir(), "assets")
	require.NoError(t, os.MkdirAll(assetDir, 0o755))
	require.NoError(t, SaveExtractor("", &Extractor{
		OASFURL:  "https://schema.oasf.outshift.com",
		AssetDir: assetDir,
	}))

	_, err := ResolveConfigured()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "provisioned")
}

func TestResolveConfiguredRemoteSucceeds(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	// A persisted RemoteAddr resolves to the remote backend with no local assets
	// (the gRPC connection is lazy, so resolution succeeds without a live server).
	require.NoError(t, SaveExtractor("", &Extractor{RemoteAddr: "oasf-sdk:5000"}))

	ext, err := ResolveConfigured()
	require.NoError(t, err)
	require.NotNil(t, ext)

	assert.NoError(t, ext.Close())
}
