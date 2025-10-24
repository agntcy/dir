// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSchema(t *testing.T) {
	t.Run("should return schema for valid version", func(t *testing.T) {
		ctx := context.Background()
		input := GetSchemaInput{Version: "0.7.0"}

		_, output, err := GetSchema(ctx, nil, input)

		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.Equal(t, "0.7.0", output.Version)
		assert.NotEmpty(t, output.Schema)
		assert.Greater(t, len(output.Schema), 100, "Schema should be a reasonable size")
		assert.NotEmpty(t, output.AvailableVersions)
	})

	t.Run("should return error for invalid version", func(t *testing.T) {
		ctx := context.Background()
		input := GetSchemaInput{Version: "99.99.99"}

		_, output, err := GetSchema(ctx, nil, input)

		require.NoError(t, err)
		assert.NotEmpty(t, output.ErrorMessage)
		assert.Contains(t, output.ErrorMessage, "Invalid version")
		assert.Empty(t, output.Schema)
	})

	t.Run("should return error for empty version", func(t *testing.T) {
		ctx := context.Background()
		input := GetSchemaInput{Version: ""}

		_, output, err := GetSchema(ctx, nil, input)

		require.NoError(t, err)
		assert.NotEmpty(t, output.ErrorMessage)
		assert.Contains(t, output.ErrorMessage, "required")
	})
}
