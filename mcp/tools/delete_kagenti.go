// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DeleteKagentiInput defines the input parameters for deleting a Kagenti agent.
type DeleteKagentiInput struct {
	// AgentName is the name of the Agent CR to delete
	AgentName string `json:"agent_name" jsonschema:"Name of the Agent CR to delete (required)"`
	// Namespace is the Kubernetes namespace where the agent is deployed
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (default: default)"`
}

// DeleteKagentiOutput defines the output of deleting an agent.
type DeleteKagentiOutput struct {
	AgentName    string `json:"agent_name,omitempty"    jsonschema:"Name of the deleted Agent CR"`
	Namespace    string `json:"namespace,omitempty"     jsonschema:"Namespace where the agent was deployed"`
	Deleted      bool   `json:"deleted"                 jsonschema:"True if the agent was successfully deleted"`
	ErrorMessage string `json:"error_message,omitempty" jsonschema:"Error message if deletion failed"`
}

// DeleteKagenti deletes a Kagenti Agent CR from Kubernetes.
func DeleteKagenti(ctx context.Context, _ *mcp.CallToolRequest, input DeleteKagentiInput) (
	*mcp.CallToolResult,
	DeleteKagentiOutput,
	error,
) {
	// Validate input
	if input.AgentName == "" {
		return nil, DeleteKagentiOutput{
			ErrorMessage: "agent_name is required",
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
		return nil, DeleteKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to create Kubernetes client: %v", err),
		}, nil
	}

	// Define the GVR for Agent resources
	gvr := schema.GroupVersionResource{
		Group:    KagentiAPIGroup,
		Version:  KagentiAPIVersion,
		Resource: KagentiAgentResource,
	}

	// Delete the resource
	err = k8sClient.Resource(gvr).Namespace(namespace).Delete(ctx, input.AgentName, metav1.DeleteOptions{})
	if err != nil {
		return nil, DeleteKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to delete Agent CR '%s' in namespace '%s': %v", input.AgentName, namespace, err),
		}, nil
	}

	return nil, DeleteKagentiOutput{
		AgentName: input.AgentName,
		Namespace: namespace,
		Deleted:   true,
	}, nil
}

