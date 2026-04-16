// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package toolhost

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

// ChatCompleter is the LLM backend for the MCP tool loop. It is satisfied today by
// OpenAI-compatible HTTP APIs (OpenAI, Azure OpenAI, Ollama, proxies, etc.). Other
// providers can be added with new implementations of this interface.
type ChatCompleter interface {
	// Model returns the model or deployment identifier sent on each request (may include a provider prefix).
	Model() string
	// ChatCompletion runs one chat completion step with the given messages and tools.
	ChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}

// newChatCompleter selects and constructs a ChatCompleter from the enricher config "model" string.
// Currently only OpenAI-compatible backends are implemented.
func newChatCompleter(model string) (ChatCompleter, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("model is empty")
	}

	return newOpenAIClientFromModel(model)
}
