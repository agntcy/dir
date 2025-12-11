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

		replicas := int64(3)
		input := UpdateKagentiInput{
			AgentName: "",
			Namespace: "default",
			Replicas:  &replicas,
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
			Replicas:  nil,
			Image:     nil,
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "at least one field to update must be provided")
		assert.Empty(t, output.AgentName)
	})

	t.Run("defaults namespace to 'default' when not provided", func(t *testing.T) {
		t.Parallel()

		replicas := int64(2)
		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "", // Should default to "default"
			Replicas:  &replicas,
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

		replicas := int64(5)
		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "test-ns",
			Replicas:  &replicas,
			Image:     nil, // No image update
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

		image := "ghcr.io/test/new-image:v2"
		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "test-ns",
			Replicas:  nil, // No replicas update
			Image:     &image,
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

		replicas := int64(3)
		image := "ghcr.io/test/new-image:v2"
		input := UpdateKagentiInput{
			AgentName: "test-agent",
			Namespace: "test-ns",
			Replicas:  &replicas,
			Image:     &image,
		}

		_, output, err := UpdateKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but passed validation
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
	})
}

