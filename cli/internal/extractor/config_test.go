// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigResolveFillsDefaults(t *testing.T) {
	got := Config{}.Resolve()
	assert.Equal(t, DefaultOASFURL, got.OASFURL)
	assert.Equal(t, DefaultAssetDir(), got.AssetDir)
}

func TestConfigResolveKeepsExplicit(t *testing.T) {
	got := Config{OASFURL: "https://local", AssetDir: "/tmp/x"}.Resolve()
	assert.Equal(t, "https://local", got.OASFURL)
	assert.Equal(t, "/tmp/x", got.AssetDir)
}

func TestDefaultAssetDir(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".agntcy", "oasf-sdk", "extractor"), DefaultAssetDir())
}

func TestConfigValidate(t *testing.T) {
	const absDir = "/tmp/assets"

	require.NoError(t, Config{OASFURL: "https://schema.oasf.outshift.com", AssetDir: absDir}.Validate())
	require.NoError(t, Config{OASFURL: "http://localhost:8080", AssetDir: absDir}.Validate())

	// OASF URL problems.
	require.Error(t, Config{OASFURL: "", AssetDir: absDir}.Validate())
	require.Error(t, Config{OASFURL: "not-a-url", AssetDir: absDir}.Validate())
	require.Error(t, Config{OASFURL: "ftp://x", AssetDir: absDir}.Validate())

	// AssetDir problems: empty or relative.
	require.Error(t, Config{OASFURL: "https://schema.oasf.outshift.com", AssetDir: ""}.Validate())
	require.Error(t, Config{OASFURL: "https://schema.oasf.outshift.com", AssetDir: "relative/dir"}.Validate())
	require.Error(t, Config{OASFURL: "https://schema.oasf.outshift.com", AssetDir: "../up"}.Validate())
}
