// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DeleteAgentInput defines the input parameters for the delete_agent prompt.
type DeleteAgentInput struct {
	AgentName string `json:"agent_name" jsonschema:"Name of the agent to delete (required)"`
	Namespace string `json:"namespace"  jsonschema:"Kubernetes namespace (default: default)"`
}

// DeleteAgent implements the delete_agent prompt.
// It deletes a deployed agent from Kubernetes.
func DeleteAgent(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	args := req.Params.Arguments

	agentName := args["agent_name"]
	if agentName == "" {
		agentName = "[User will provide agent name]"
	}

	namespace := args["namespace"]
	if namespace == "" {
		namespace = "default"
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
Delete a deployed agent from Kubernetes.

Agent name: %s
Namespace: %s

Use agntcy_kagenti_delete with:
- agent_name: "%s"
- namespace: "%s"

Confirm the deletion to the user.
	`), agentName, namespace, agentName, namespace)

	return &mcp.GetPromptResult{
		Description: "Delete a deployed agent from Kubernetes",
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

