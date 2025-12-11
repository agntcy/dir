// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// KagentiAPIGroup is the API group for Kagenti resources.
	KagentiAPIGroup = "agent.kagenti.dev"
	// KagentiAPIVersion is the API version for Kagenti resources.
	KagentiAPIVersion = "v1alpha1"
	// KagentiAgentResource is the resource name for Agent resources.
	KagentiAgentResource = "agents"
)

// DeployKagentiInput defines the input parameters for deploying an agent via Kagenti.
type DeployKagentiInput struct {
	// AgentJSON is the marshalled Kagenti Agent CR JSON string
	AgentJSON string `json:"agent_json" jsonschema:"Marshalled Kagenti Agent CR as JSON string (required)"`
	// Namespace is the Kubernetes namespace to deploy to
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace to deploy to (default: default)"`
	// Replicas is the number of pod replicas
	Replicas int64 `json:"replicas,omitempty" jsonschema:"Number of pod replicas (default: 1)"`
}

// DeployKagentiOutput defines the output of deploying an agent.
type DeployKagentiOutput struct {
	AgentName    string `json:"agent_name,omitempty"    jsonschema:"Name of the created/updated Agent CR"`
	Namespace    string `json:"namespace,omitempty"     jsonschema:"Namespace where the agent was deployed"`
	Created      bool   `json:"created"                 jsonschema:"True if created, false if updated"`
	ErrorMessage string `json:"error_message,omitempty" jsonschema:"Error message if deployment failed"`
}

// DeployKagenti deploys a Kagenti Agent CR to Kubernetes.
// The Agent CR should be provided as a marshalled JSON string.
func DeployKagenti(ctx context.Context, _ *mcp.CallToolRequest, input DeployKagentiInput) (
	*mcp.CallToolResult,
	DeployKagentiOutput,
	error,
) {
	// Validate input
	if input.AgentJSON == "" {
		return nil, DeployKagentiOutput{
			ErrorMessage: "agent_json is required",
		}, nil
	}

	// Unmarshal JSON into unstructured object
	var obj unstructured.Unstructured
	if err := json.Unmarshal([]byte(input.AgentJSON), &obj.Object); err != nil {
		return nil, DeployKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to unmarshal Agent JSON: %v", err),
		}, nil
	}

	// Validate it's a Kagenti Agent
	gvk := obj.GroupVersionKind()
	if gvk.Group != KagentiAPIGroup || gvk.Version != KagentiAPIVersion || gvk.Kind != "Agent" {
		return nil, DeployKagentiOutput{
			ErrorMessage: fmt.Sprintf("Invalid resource type: expected %s/%s/Agent, got %s/%s/%s",
				KagentiAPIGroup, KagentiAPIVersion, gvk.Group, gvk.Version, gvk.Kind),
		}, nil
	}

	// Determine namespace (default to "default")
	namespace := input.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Determine replicas (default to 1)
	replicas := input.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	// Set namespace on the object
	obj.SetNamespace(namespace)

	// Set replicas in spec
	if err := unstructured.SetNestedField(obj.Object, replicas, "spec", "replicas"); err != nil {
		return nil, DeployKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to set replicas: %v", err),
		}, nil
	}

	agentName := obj.GetName()
	if agentName == "" {
		return nil, DeployKagentiOutput{
			ErrorMessage: "Agent CR must have a name",
		}, nil
	}

	// Apply to Kubernetes
	created, err := applyUnstructured(ctx, &obj, namespace)
	if err != nil {
		return nil, DeployKagentiOutput{
			ErrorMessage: fmt.Sprintf("Failed to apply Agent CR to Kubernetes: %v", err),
		}, nil
	}

	return nil, DeployKagentiOutput{
		AgentName: agentName,
		Namespace: namespace,
		Created:   created,
	}, nil
}

// applyUnstructured applies an unstructured object to Kubernetes.
// Returns true if created, false if updated.
func applyUnstructured(ctx context.Context, obj *unstructured.Unstructured, namespace string) (bool, error) {
	// Get Kubernetes client
	k8sClient, err := getK8sClient()
	if err != nil {
		return false, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Define the GVR for Agent resources
	gvr := schema.GroupVersionResource{
		Group:    KagentiAPIGroup,
		Version:  KagentiAPIVersion,
		Resource: KagentiAgentResource,
	}

	// Try to get existing resource first
	existing, err := k8sClient.Resource(gvr).Namespace(namespace).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err == nil {
		// Resource exists, update it
		obj.SetResourceVersion(existing.GetResourceVersion())
		_, err = k8sClient.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to update Agent CR: %w", err)
		}
		return false, nil
	}

	// Resource doesn't exist, create it
	_, err = k8sClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to create Agent CR: %w", err)
	}
	return true, nil
}

// getK8sClient creates a dynamic Kubernetes client.
func getK8sClient() (dynamic.Interface, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
		}
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return dynamicClient, nil
}
