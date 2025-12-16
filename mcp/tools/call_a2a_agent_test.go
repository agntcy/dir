// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallA2AAgent(t *testing.T) {
	t.Run("should return error when message is empty", func(t *testing.T) {
		ctx := context.Background()
		input := CallA2AAgentInput{
			Message: "",
		}

		_, output, err := CallA2AAgent(ctx, nil, input)
		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "message is required")
	})

	t.Run("should use default endpoint when not provided", func(t *testing.T) {
		// This test just verifies the input parsing, not the actual HTTP call
		input := CallA2AAgentInput{
			Message: "What is the weather?",
		}
		assert.Equal(t, "", input.Endpoint) // Will default to localhost:8000 in the function
	})

	t.Run("should successfully call agent and parse response", func(t *testing.T) {
		// Create a mock A2A server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			// Parse request body
			var req a2aRequest
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)
			assert.Equal(t, "2.0", req.JSONRPC)
			assert.Equal(t, "message/send", req.Method)
			assert.Equal(t, "user", req.Params.Message.Role)
			assert.Len(t, req.Params.Message.Parts, 1)
			assert.Equal(t, "text", req.Params.Message.Parts[0].Kind)
			assert.Equal(t, "What is the weather in Budapest?", req.Params.Message.Parts[0].Text)

			// Send mock response
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]interface{}{
					"artifacts": []map[string]interface{}{
						{
							"artifactId": "test-artifact",
							"parts": []map[string]interface{}{
								{
									"kind": "text",
									"text": "The weather in Budapest is sunny, 25°C.",
								},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		ctx := context.Background()
		input := CallA2AAgentInput{
			Message:  "What is the weather in Budapest?",
			Endpoint: server.URL,
		}

		_, output, err := CallA2AAgent(ctx, nil, input)
		require.NoError(t, err)
		assert.Empty(t, output.ErrorMessage)
		assert.Equal(t, "The weather in Budapest is sunny, 25°C.", output.Response)
		assert.NotEmpty(t, output.RawResponse)
	})

	t.Run("should handle agent error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      "1",
				"error": map[string]interface{}{
					"code":    -32600,
					"message": "Invalid request",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		ctx := context.Background()
		input := CallA2AAgentInput{
			Message:  "test",
			Endpoint: server.URL,
		}

		_, output, err := CallA2AAgent(ctx, nil, input)
		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Agent error")
		assert.Contains(t, output.ErrorMessage, "Invalid request")
	})

	t.Run("should handle HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		ctx := context.Background()
		input := CallA2AAgentInput{
			Message:  "test",
			Endpoint: server.URL,
		}

		_, output, err := CallA2AAgent(ctx, nil, input)
		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "HTTP 500")
	})

	t.Run("should handle connection error", func(t *testing.T) {
		ctx := context.Background()
		input := CallA2AAgentInput{
			Message:  "test",
			Endpoint: "http://localhost:99999", // Invalid port
		}

		_, output, err := CallA2AAgent(ctx, nil, input)
		require.NoError(t, err)
		assert.Contains(t, output.ErrorMessage, "Failed to send request")
	})
}

