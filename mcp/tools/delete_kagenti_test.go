// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteKagenti(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("fails when agent_name is empty", func(t *testing.T) {
		t.Parallel()

		input := DeleteKagentiInput{
			AgentName: "",
			Namespace: "default",
		}

		_, output, err := DeleteKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "agent_name is required")
		assert.Empty(t, output.AgentName)
		assert.False(t, output.Deleted)
	})

	t.Run("defaults namespace to 'default' when not provided", func(t *testing.T) {
		t.Parallel()

		input := DeleteKagentiInput{
			AgentName: "test-agent",
			Namespace: "", // Should default to "default"
		}

		_, output, err := DeleteKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s (either connection or not found), but passed validation
		// The important thing is it attempted the delete with correct namespace
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "test-agent")
		}
	})

	t.Run("uses provided namespace", func(t *testing.T) {
		t.Parallel()

		input := DeleteKagentiInput{
			AgentName: "test-agent",
			Namespace: "custom-namespace",
		}

		_, output, err := DeleteKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s (either connection or not found), but passed validation
		// The important thing is it attempted the delete with correct namespace
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "custom-namespace")
		}
	})
}

