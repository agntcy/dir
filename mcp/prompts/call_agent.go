// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CallAgentInput defines the input parameters for the call_agent prompt.
type CallAgentInput struct {
	Message  string `json:"message"  jsonschema:"The question or message to send to the agent (required)"`
	Endpoint string `json:"endpoint" jsonschema:"The A2A agent endpoint URL (default: http://localhost:8000)"`
}

// CallAgent implements the call_agent prompt.
// It sends a message to a deployed A2A agent and returns the response.
func CallAgent(_ context.Context, req *mcp.GetPromptRequest) (
	*mcp.GetPromptResult,
	error,
) {
	args := req.Params.Arguments

	message := args["message"]
	if message == "" {
		message = "[User will provide their question]"
	}

	endpoint := args["endpoint"]
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	promptText := fmt.Sprintf(strings.TrimSpace(`
Send a message to a deployed A2A agent and return the response.

Message: "%s"
Endpoint: %s

Use agntcy_a2a_call with:
- message: "%s"
- endpoint: "%s"

Return the agent's response to the user.
	`), message, endpoint, message, endpoint)

	return &mcp.GetPromptResult{
		Description: "Call a deployed A2A agent with a message",
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

