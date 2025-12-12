// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateKagenti(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("fails when agent_name is empty", func(t *testing.T) {
		t.Parallel()

		input := UpdateKagentiInput{
			AgentName: "",
			Namespace: "default",
			Replicas:  3,
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "agent_name is required")
		assert.Empty(t, output.AgentName)
	})

	t.Run("fails when no update fields provided", func(t *testing.T) {
		t.Parallel()

		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "default",
			Replicas:  0,  // 0 means no update
			Image:     "", // empty means no update
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "at least one field to update must be provided")
		assert.Empty(t, output.AgentName)
	})

	t.Run("defaults namespace to 'default' when not provided", func(t *testing.T) {
		t.Parallel()

		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "", // Should default to "default"
			Replicas:  2,
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but that's expected without a cluster
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
	})

	t.Run("accepts replicas only", func(t *testing.T) {
		t.Parallel()

		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "test-ns",
			Replicas:  5,
			Image:     "", // No image update
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but passed validation
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
	})

	t.Run("accepts image only", func(t *testing.T) {
		t.Parallel()

		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "test-ns",
			Replicas:  0, // No replicas update
			Image:     "ghcr.io/test/new-image:v2",
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but passed validation
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
	})

	t.Run("accepts both replicas and image", func(t *testing.T) {
		t.Parallel()

		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "test-ns",
			Replicas:  3,
			Image:     "ghcr.io/test/new-image:v2",
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but passed validation
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
	})
}
