// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package toolhost runs the enricher agent loop: chat completions (via [ChatCompleter])
// with MCP tools over stdio (no third-party MCP host runtime).
package toolhost

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	openai "github.com/sashabaranov/go-openai"
)

// Host implements MCP tools + LLM tool-calling loop used by the importer enricher.
type Host struct {
	mcpClient *client.Client
	llm       ChatCompleter
	tools     []openai.Tool
	maxSteps  int
}

// NewFromConfigFile loads enricher JSON, starts the MCP stdio server, and prepares the LLM client.
func NewFromConfigFile(ctx context.Context, configPath string) (*Host, error) {
	cfg, err := LoadFileConfig(configPath)
	if err != nil {
		return nil, err
	}

	srv, err := cfg.DirMCPServer()
	if err != nil {
		return nil, err
	}

	extraEnv := srv.ExtraEnv()

	mcpCli, err := client.NewStdioMCPClientWithOptions(srv.Command, extraEnv, srv.Args)
	if err != nil {
		return nil, fmt.Errorf("start MCP stdio client: %w", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "dir-importer-enricher",
		Version: "1.0.0",
	}

	if _, err := mcpCli.Initialize(ctx, initReq); err != nil {
		_ = mcpCli.Close()

		return nil, fmt.Errorf("MCP initialize: %w", err)
	}

	listReq := mcp.ListToolsRequest{}

	toolsRes, err := mcpCli.ListTools(ctx, listReq)
	if err != nil {
		_ = mcpCli.Close()

		return nil, fmt.Errorf("MCP list tools: %w", err)
	}

	openaiTools, err := mcpToolsToOpenAI(toolsRes.Tools)
	if err != nil {
		_ = mcpCli.Close()

		return nil, err
	}

	if len(openaiTools) == 0 {
		_ = mcpCli.Close()

		return nil, fmt.Errorf("no MCP tools available from MCP server")
	}

	llm, err := newChatCompleter(cfg.Model)
	if err != nil {
		_ = mcpCli.Close()

		return nil, err
	}

	return &Host{
		mcpClient: mcpCli,
		llm:       llm,
		tools:     openaiTools,
		maxSteps:  cfg.MaxSteps,
	}, nil
}

func mcpToolsToOpenAI(tools []mcp.Tool) ([]openai.Tool, error) {
	out := make([]openai.Tool, 0, len(tools))
	for _, t := range tools {
		params, err := toolParametersJSON(t)
		if err != nil {
			return nil, fmt.Errorf("tool %q: params schema: %w", t.Name, err)
		}

		out = append(out, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  params,
			},
		})
	}

	return out, nil
}

func toolParametersJSON(t mcp.Tool) (json.RawMessage, error) {
	if len(t.RawInputSchema) > 0 {
		return t.RawInputSchema, nil
	}

	b, err := json.Marshal(t.InputSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal MCP tool input schema: %w", err)
	}

	if len(b) == 0 || string(b) == "null" {
		return json.RawMessage(`{"type":"object"}`), nil
	}

	return b, nil
}

// Close closes the MCP client.
func (h *Host) Close() error {
	if h == nil || h.mcpClient == nil {
		return nil
	}

	if err := h.mcpClient.Close(); err != nil {
		return fmt.Errorf("close MCP client: %w", err)
	}

	return nil
}

// ClearSession is a no-op: each Prompt already starts a fresh message list.
func (h *Host) ClearSession() {}

// Prompt runs a user message through the tool loop and returns the assistant text.
func (h *Host) Prompt(ctx context.Context, message string) (string, error) {
	return h.prompt(ctx, message, nil, nil, nil)
}

// PromptWithCallbacks runs Prompt with optional callbacks (streaming callback is not used yet).
func (h *Host) PromptWithCallbacks(
	ctx context.Context,
	message string,
	onToolCall func(name, args string),
	onToolResult func(name, args, result string, isError bool),
	onChunk func(chunk string),
) (string, error) {
	_ = onChunk

	return h.prompt(ctx, message, onToolCall, onToolResult, nil)
}

func (h *Host) prompt(
	ctx context.Context,
	message string,
	onToolCall func(name, args string),
	onToolResult func(name, args, result string, isError bool),
	onStreaming func(chunk string),
) (string, error) {
	_ = onStreaming

	if h == nil || h.llm == nil || h.mcpClient == nil {
		return "", errors.New("tool host is not initialized")
	}

	msgs := []openai.ChatCompletionMessage{{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	}}

	for range h.maxSteps {
		req := openai.ChatCompletionRequest{
			Model:    h.llm.Model(),
			Messages: msgs,
			Tools:    h.tools,
		}

		resp, err := h.llm.ChatCompletion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("chat completion: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("chat completion: empty choices")
		}

		msg := resp.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			return strings.TrimSpace(msg.Content), nil
		}

		msgs = append(msgs, msg)

		for _, tc := range msg.ToolCalls {
			if onToolCall != nil {
				onToolCall(tc.Function.Name, tc.Function.Arguments)
			}

			toolText, _ := h.callMCPTool(ctx, tc, onToolResult)
			msgs = append(msgs, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    toolText,
				Name:       tc.Function.Name,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("exceeded max tool/LLM steps (%d)", h.maxSteps)
}

func (h *Host) callMCPTool(
	ctx context.Context,
	tc openai.ToolCall,
	onToolResult func(name, args, result string, isError bool),
) (string, bool) {
	if tc.Type != "" && tc.Type != openai.ToolTypeFunction {
		errText := fmt.Sprintf("unsupported tool call type %q", tc.Type)
		if onToolResult != nil {
			onToolResult(tc.Function.Name, tc.Function.Arguments, errText, true)
		}

		return errText, true
	}

	var args any
	if tc.Function.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
			errText := fmt.Sprintf("invalid tool arguments JSON: %v", err)
			if onToolResult != nil {
				onToolResult(tc.Function.Name, tc.Function.Arguments, errText, true)
			}

			return errText, true
		}
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = tc.Function.Name
	req.Params.Arguments = args

	res, err := h.mcpClient.CallTool(ctx, req)
	if err != nil {
		errText := fmt.Sprintf("MCP tool error: %v", err)
		if onToolResult != nil {
			onToolResult(tc.Function.Name, tc.Function.Arguments, errText, true)
		}

		return errText, true
	}

	text := formatToolResult(res)

	isErr := res.IsError
	if onToolResult != nil {
		onToolResult(tc.Function.Name, tc.Function.Arguments, text, isErr)
	}

	return text, isErr
}

func formatToolResult(res *mcp.CallToolResult) string {
	if res == nil {
		return ""
	}

	var b strings.Builder

	for _, c := range res.Content {
		switch t := c.(type) {
		case mcp.TextContent:
			b.WriteString(t.Text)
		default:
			raw, err := json.Marshal(c)
			if err != nil {
				fmt.Fprintf(&b, "%v", c)
			} else {
				b.Write(raw)
			}
		}
	}

	if b.Len() > 0 {
		return b.String()
	}

	if res.StructuredContent != nil {
		raw, err := json.Marshal(res.StructuredContent)
		if err != nil {
			return fmt.Sprintf("%v", res.StructuredContent)
		}

		return string(raw)
	}

	return ""
}
