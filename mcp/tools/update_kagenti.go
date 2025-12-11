// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// UpdateKagentiInput defines the input parameters for updating a Kagenti agent.
type UpdateKagentiInput struct {
	// AgentName is the name of the Agent CR to update
	AgentName string `json:"agent_name" jsonschema:"Name of the Agent CR to update (required)"`
	// Namespace is the Kubernetes namespace where the agent is deployed
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (default: default)"`
	// Replicas is the new number of pod replicas (only updated if provided)
	Replicas *int64 `json:"replicas,omitempty" jsonschema:"New number of pod replicas (optional)"`
	// Image is the new container image (only updated if provided)
	Image *string `json:"image,omitempty" jsonschema:"New container image URL (optional)"`
}

// UpdateKagentiOutput defines the output of updating an agent.
type UpdateKagentiOutput struct {
	AgentName      string `json:"agent_name,omitempty"      jsonschema:"Name of the updated Agent CR"`
	Namespace      string `json:"namespace,omitempty"       jsonschema:"Namespace where the agent is deployed"`
	UpdatedFields  string `json:"updated_fields,omitempty"  jsonschema:"Comma-separated list of fields that were updated"`
	ErrorMessage   string `json:"error_message,omitempty"   jsonschema:"Error message if update failed"`
}

// UpdateKagenti updates an existing Kagenti Agent CR in Kubernetes.
// Only the specified fields are updated (replicas, image).
func UpdateKagenti(ctx context.Context, _ *mcp.CallToolRequest, input UpdateKagentiInput) (
	*mcp.CallToolResult,
	UpdateKagentiOutput,
	error,
) {
	// Validate input
	if input.AgentName == "" {
		return nil, UpdateKagentiOutput{
			ErrorMessage: "agent_name is required",
		}, nil
	}

	// Check that at least one field to update is provided
	if input.Replicas == nil && input.Image == nil {
		return nil, UpdateKagentiOutput{
			ErrorMessage: "at least one field to update must be provided (replicas or image)",
		}, nil
	}

	// Determine namespace (default to "default")
	namespace := input.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Get Kubernetes client
	k8sClient, err := getK8sClient()
	if err != nil {
		return nil, UpdateKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to create Kubernetes client: %v", err),
		}, nil
	}

	// Define the GVR for Agent resources
	gvr := schema.GroupVersionResource{
		Group:    KagentiAPIGroup,
		Version:  KagentiAPIVersion,
		Resource: KagentiAgentResource,
	}

	// Get existing resource
	existing, err := k8sClient.Resource(gvr).Namespace(namespace).Get(ctx, input.AgentName, metav1.GetOptions{})
	if err != nil {
		return nil, UpdateKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to get Agent CR '%s' in namespace '%s': %v", input.AgentName, namespace, err),
		}, nil
	}

	// Track which fields were updated
	var updatedFields []string

	// Update replicas if provided
	if input.Replicas != nil {
		if err := unstructured.SetNestedField(existing.Object, *input.Replicas, "spec", "replicas"); err != nil {
			return nil, UpdateKagentiOutput{
				ErrorMessage: fmt.Sprintf("Failed to set replicas: %v", err),
			}, nil
		}
		updatedFields = append(updatedFields, "replicas")
	}

	// Update image if provided
	if input.Image != nil {
		// Update imageSource.image
		if err := unstructured.SetNestedField(existing.Object, *input.Image, "spec", "imageSource", "image"); err != nil {
			return nil, UpdateKagentiOutput{
				ErrorMessage: fmt.Sprintf("Failed to set imageSource.image: %v", err),
			}, nil
		}

		// Also update the container image in podTemplateSpec if it exists
		containers, found, err := unstructured.NestedSlice(existing.Object, "spec", "podTemplateSpec", "spec", "containers")
		if err == nil && found && len(containers) > 0 {
			// Update the first container's image
			if container, ok := containers[0].(map[string]interface{}); ok {
				container["image"] = *input.Image
				containers[0] = container
				if err := unstructured.SetNestedSlice(existing.Object, containers, "spec", "podTemplateSpec", "spec", "containers"); err != nil {
					return nil, UpdateKagentiOutput{
						ErrorMessage: fmt.Sprintf("Failed to set container image: %v", err),
					}, nil
				}
			}
		}
		updatedFields = append(updatedFields, "image")
	}

	// Apply the update
	_, err = k8sClient.Resource(gvr).Namespace(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return nil, UpdateKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to update Agent CR: %v", err),
		}, nil
	}

	// Build updated fields string
	updatedFieldsStr := ""
	for i, field := range updatedFields {
		if i > 0 {
			updatedFieldsStr += ", "
		}
		updatedFieldsStr += field
	}

	return nil, UpdateKagentiOutput{
		AgentName:     input.AgentName,
		Namespace:     namespace,
		UpdatedFields: updatedFieldsStr,
	}, nil
}

