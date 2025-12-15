// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeployAgentInput defines the input parameters for the deploy_agent prompt.
type DeployAgentInput struct {
	Description string `json:"description" jsonschema:"Natural language description of what kind of agent you need (e.g., 'I need an agent that can tell me the weather', 'find me a code review assistant') (required)"`
	Namespace   string `json:"namespace"   jsonschema:"Kubernetes namespace to deploy to (default: 'default')"`
	Replicas    string `json:"replicas"    jsonschema:"Number of pod replicas to deploy (default: '1')"`
}

// DeployAgent implements the deploy_agent prompt.
// It guides users through the complete workflow of searching, pulling, exporting, and deploying an agent.
func DeployAgent(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	// Parse arguments from the request
	args := req.Params.Arguments

	description := args["description"]
	if description == "" {
		description = "[User will describe what agent they need]"
	}

	namespace := args["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	replicas := args["replicas"]
	if replicas == "" {
		replicas = "1"
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
Find and deploy an agent from the Directory to Kubernetes.

User request: "%s"
Target namespace: %s
Replicas: %s

## Workflow

1. **Search**: Use agntcy_dir_search_local to find agents matching the user's request
2. **Pull**: Use agntcy_dir_pull_record with the CID from search results
3. **Export**: Use agntcy_oasf_export_record with target_format "kagenti"
4. **Deploy**: Use agntcy_kagenti_deploy to namespace "%s" with %s replica(s)

## After Deployment

Provide the user with:
- kubectl get agents -n %s
- kubectl port-forward svc/<agent-name> 8000:8000 -n %s

Start by searching for agents that match: "%s"
	`), description, namespace, replicas,
		namespace, replicas,
		namespace, namespace,
		description)

	return &mcp.GetPromptResult{
		Description: "Deploy an agent from the Directory to a Kubernetes cluster via Kagenti",
		Messages: []*mcp.PromptMessage{
			{
				Role: "user",
				Content: &mcp.TextContent{
					Text: promptText,
				},
			},
		},
	}, nil
}
