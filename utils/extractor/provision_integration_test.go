// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//go:build extractor_integration

package extractor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// Run with: go test -tags extractor_integration ./internal/extractor/ -run TestProvisionSmokeCheck
// Downloads the ~89 MB model and fetches the taxonomy from DefaultOASFURL.
func TestProvisionSmokeCheckIntegration(t *testing.T) {
	cfg := Config{AssetDir: t.TempDir()}.Resolve()

	require.NoError(t, Provision(context.Background(), cfg))
	require.True(t, IsProvisioned(cfg))
	require.NoError(t, SmokeCheck(context.Background(), cfg))
}
