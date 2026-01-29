// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}

func TestListVersions(t *testing.T) {
	// ListVersions doesn't require schema URL, but set it for consistency
	t.Setenv("OASF_API_VALIDATION_SCHEMA_URL", "https://schema.oasf.outshift.com")

	t.Run("should return available versions", func(t *testing.T) {
		ctx := context.Background()
		input := ListVersionsInput{}

		_, output, err := ListVersions(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.NotEmpty(t, output.AvailableVersions)
		assert.NotEmpty(t, output.DefaultVersion, "Default version should be set")
		assert.Positive(t, output.Count)
		assert.Len(t, output.AvailableVersions, output.Count)
		// Default version should be in the available versions list
		assert.Contains(t, output.AvailableVersions, output.DefaultVersion, "Default version should be in available versions")
	})

	t.Run("should include known versions", func(t *testing.T) {
		ctx := context.Background()
		input := ListVersionsInput{}

		_, output, err := ListVersions(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.AvailableVersions, "0.7.0")
		// 1.0.0-rc.1 may not be available in all test environments, so we check but don't fail
		if !contains(output.AvailableVersions, "1.0.0-rc.1") {
			t.Logf("Note: 1.0.0-rc.1 not found in available versions: %v", output.AvailableVersions)
		}

		assert.NotEmpty(t, output.DefaultVersion, "Default version should be set")
	})
}
