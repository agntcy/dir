// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CallA2AAgentInput defines the input parameters for calling an A2A agent.
type CallA2AAgentInput struct {
	// Message is the text message to send to the agent
	Message string `json:"message" jsonschema:"The message/question to send to the agent (required)"`
	// Endpoint is the URL of the A2A agent (default: http://localhost:8000)
	Endpoint string `json:"endpoint,omitempty" jsonschema:"The A2A agent endpoint URL (default: http://localhost:8000)"`
}

// CallA2AAgentOutput defines the output of calling an A2A agent.
type CallA2AAgentOutput struct {
	Response     string `json:"response,omitempty"      jsonschema:"The agent's response text"`
	RawResponse  string `json:"raw_response,omitempty"  jsonschema:"The full JSON-RPC response"`
	ErrorMessage string `json:"error_message,omitempty" jsonschema:"Error message if the call failed"`
}

// a2aRequest represents the JSON-RPC request structure for A2A protocol.
type a2aRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  a2aParams   `json:"params"`
}

type a2aParams struct {
	Message a2aMessage `json:"message"`
}

type a2aMessage struct {
	Role      string       `json:"role"`
	MessageID string       `json:"messageId"`
	Parts     []a2aPart    `json:"parts"`
}

type a2aPart struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// a2aResponse represents the JSON-RPC response structure.
type a2aResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  *a2aResult  `json:"result,omitempty"`
	Error   *a2aError   `json:"error,omitempty"`
}

type a2aResult struct {
	Artifacts []a2aArtifact `json:"artifacts,omitempty"`
}

type a2aArtifact struct {
	ArtifactID string    `json:"artifactId"`
	Parts      []a2aPart `json:"parts,omitempty"`
}

type a2aError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CallA2AAgent sends a message to an A2A agent and returns the response.
func CallA2AAgent(ctx context.Context, _ *mcp.CallToolRequest, input CallA2AAgentInput) (
	*mcp.CallToolResult,
	CallA2AAgentOutput,
	error,
) {
	// Validate input
	if input.Message == "" {
		return nil, CallA2AAgentOutput{
			ErrorMessage: "message is required",
		}, nil
	}

	// Set default endpoint
	endpoint := input.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:8000"
	}

	// Build the A2A request
	req := a2aRequest{
		JSONRPC: "2.0",
		ID:      uuid.New().String(),
		Method:  "message/send",
		Params: a2aParams{
			Message: a2aMessage{
				Role:      "user",
				MessageID: uuid.New().String(),
				Parts: []a2aPart{
					{
						Kind: "text",
						Text: input.Message,
					},
				},
			},
		},
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Failed to marshal request: %v", err),
		}, nil
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Failed to create HTTP request: %v", err),
		}, nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request with timeout
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Failed to send request to agent: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Failed to read response: %v", err),
		}, nil
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Agent returned HTTP %d: %s", resp.StatusCode, string(respBody)),
		}, nil
	}

	// Parse JSON-RPC response
	var a2aResp a2aResponse
	if err := json.Unmarshal(respBody, &a2aResp); err != nil {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Failed to parse response: %v", err),
			RawResponse:  string(respBody),
		}, nil
	}

	// Check for JSON-RPC error
	if a2aResp.Error != nil {
		return nil, CallA2AAgentOutput{
			ErrorMessage: fmt.Sprintf("Agent error (%d): %s", a2aResp.Error.Code, a2aResp.Error.Message),
			RawResponse:  string(respBody),
		}, nil
	}

	// Extract response text from artifacts
	var responseText string
	if a2aResp.Result != nil && len(a2aResp.Result.Artifacts) > 0 {
		for _, artifact := range a2aResp.Result.Artifacts {
			for _, part := range artifact.Parts {
				if part.Kind == "text" {
					if responseText != "" {
						responseText += "\n"
					}
					responseText += part.Text
				}
			}
		}
	}

	return nil, CallA2AAgentOutput{
		Response:    responseText,
		RawResponse: string(respBody),
	}, nil
}

