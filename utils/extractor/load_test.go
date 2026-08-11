// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"testing"

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
