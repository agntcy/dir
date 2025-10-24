// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListVersions(t *testing.T) {
	t.Run("should return available versions", func(t *testing.T) {
		ctx := context.Background()
		input := ListVersionsInput{}

		_, output, err := ListVersions(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.NotEmpty(t, output.AvailableVersions)
		assert.Positive(t, output.Count)
		assert.Len(t, output.AvailableVersions, output.Count)
	})

	t.Run("should include known versions", func(t *testing.T) {
		ctx := context.Background()
		input := ListVersionsInput{}

		_, output, err := ListVersions(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.AvailableVersions, "0.7.0")
		assert.Contains(t, output.AvailableVersions, "0.3.1")
	})
}
