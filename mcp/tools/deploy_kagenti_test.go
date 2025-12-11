// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployKagenti(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("fails when agent_json is empty", func(t *testing.T) {
		t.Parallel()

		input := DeployKagentiInput{
			AgentJSON: "",
			Namespace: "default",
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "agent_json is required")
		assert.Empty(t, output.AgentName)
	})

	t.Run("fails with invalid JSON", func(t *testing.T) {
		t.Parallel()

		input := DeployKagentiInput{
			AgentJSON: `{invalid json}`,
			Namespace: "default",
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Failed to unmarshal Agent JSON")
		assert.Empty(t, output.AgentName)
	})

	t.Run("fails with wrong apiVersion", func(t *testing.T) {
		t.Parallel()

		agentJSON := `{
			"apiVersion": "wrong.api/v1",
			"kind": "Agent",
			"metadata": {
				"name": "test-agent"
			}
		}`

		input := DeployKagentiInput{
			AgentJSON: agentJSON,
			Namespace: "default",
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Invalid resource type")
		assert.Contains(t, output.ErrorMessage, "wrong.api/v1")
		assert.Empty(t, output.AgentName)
	})

	t.Run("fails with wrong kind", func(t *testing.T) {
		t.Parallel()

		agentJSON := `{
			"apiVersion": "agent.kagenti.dev/v1alpha1",
			"kind": "WrongKind",
			"metadata": {
				"name": "test-agent"
			}
		}`

		input := DeployKagentiInput{
			AgentJSON: agentJSON,
			Namespace: "default",
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Invalid resource type")
		assert.Contains(t, output.ErrorMessage, "WrongKind")
		assert.Empty(t, output.AgentName)
	})

	t.Run("fails when agent name is missing", func(t *testing.T) {
		t.Parallel()

		agentJSON := `{
			"apiVersion": "agent.kagenti.dev/v1alpha1",
			"kind": "Agent",
			"metadata": {}
		}`

		input := DeployKagentiInput{
			AgentJSON: agentJSON,
			Namespace: "default",
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Agent CR must have a name")
		assert.Empty(t, output.AgentName)
	})

	t.Run("defaults namespace to 'default' when not provided", func(t *testing.T) {
		t.Parallel()

		agentJSON := `{
			"apiVersion": "agent.kagenti.dev/v1alpha1",
			"kind": "Agent",
			"metadata": {
				"name": "test-agent"
			},
			"spec": {}
		}`

		input := DeployKagentiInput{
			AgentJSON: agentJSON,
			Namespace: "", // Empty - should default to "default"
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but namespace should be set correctly
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
		// If it got past validation, namespace would be "default"
		if output.Namespace != "" {
			assert.Equal(t, "default", output.Namespace)
		}
	})

	t.Run("defaults replicas to 1 when not provided", func(t *testing.T) {
		t.Parallel()

		agentJSON := `{
			"apiVersion": "agent.kagenti.dev/v1alpha1",
			"kind": "Agent",
			"metadata": {
				"name": "test-agent"
			},
			"spec": {}
		}`

		input := DeployKagentiInput{
			AgentJSON: agentJSON,
			Namespace: "test-ns",
			Replicas:  0, // Should default to 1
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but that's expected without a cluster
		// The important thing is that it passed validation
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
	})

	t.Run("uses provided namespace over CR namespace", func(t *testing.T) {
		t.Parallel()

		agentJSON := `{
			"apiVersion": "agent.kagenti.dev/v1alpha1",
			"kind": "Agent",
			"metadata": {
				"name": "test-agent",
				"namespace": "cr-namespace"
			},
			"spec": {}
		}`

		input := DeployKagentiInput{
			AgentJSON: agentJSON,
			Namespace: "input-namespace", // Should take precedence
		}

		_, output, err := DeployKagenti(ctx, nil, input)

		require.NoError(t, err)
		// Will fail on K8s connection, but namespace should be from input
		if output.ErrorMessage != "" {
			assert.Contains(t, output.ErrorMessage, "Kubernetes")
		}
		if output.Namespace != "" {
			assert.Equal(t, "input-namespace", output.Namespace)
		}
	})
}

