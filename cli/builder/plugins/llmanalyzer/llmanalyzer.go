// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package llmanalyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/agntcy/dir/cli/types"
)

const (
	PluginName    = "llmanalyzer"
	PluginVersion = "v0.0.0"

	maxRetries = 3
	prompt     = `You are a source code analyzer that must output ONLY raw JSON.

CRITICAL: Your response must only contain a single valid, parseable JSON object - no markdown, no code blocks, no explanations, no additional text.

Output Specification:
{
    "purpose": "<string>",  // Purpose in 1-2 sentences
    "workflows": [          // Array of core workflows
        "<string>",
        "<string>"
    ]
}

Violation Examples (DO NOT DO THESE):
- Do not add any markdown formatting
- Do not add explanatory text before or after the JSON
- Do not add comments within the JSON
- Do not include any whitespace before the opening {

Analyze the provided codebase focusing only on business logic and core workflows. Return the analysis as a single JSON object matching the exact structure above.`
)

type ExtensionSpecs struct {
	Purpose   string   `json:"purpose"`
	Workflows []string `json:"workflows"`
}

type llmanalyzer struct {
	cfg         *Config
	fsPath      string
	ignorePaths []string
	client      *azopenai.Client
}

func New(fsPath string, ignorePaths []string) (types.Builder, error) {
	cfg, err := NewConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load LLMAnalyzer configuration: %w", err)
	}

	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate LLMAnalyzer configuration: %w", err)
	}

	keyCredential := azcore.NewKeyCredential(cfg.AzureOpenAIAPIKey)

	client, err := azopenai.NewClientWithKeyCredential(cfg.AzureOpenAIAPIEndpoint, keyCredential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI client: %w", err)
	}

	return &llmanalyzer{
		cfg:         cfg,
		fsPath:      fsPath,
		ignorePaths: ignorePaths,
		client:      client,
	}, nil
}

func (l *llmanalyzer) Build(ctx context.Context) ([]*types.AgentExtension, error) {
	filesContent, err := concatenateFiles(l.fsPath, l.ignorePaths, []string{".py", ".yml", ".yaml"})
	if err != nil {
		return nil, fmt.Errorf("failed to concatenate files: %w", err)
	}

	messages := []azopenai.ChatRequestMessageClassification{
		&azopenai.ChatRequestSystemMessage{Content: azopenai.NewChatRequestSystemMessageContent(prompt)},
		&azopenai.ChatRequestUserMessage{Content: azopenai.NewChatRequestUserMessageContent(filesContent)},
	}

	var specs ExtensionSpecs

	var lastErr error

	for attempt := range maxRetries {
		resp, err := l.client.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
			Messages:       messages,
			DeploymentName: to.Ptr(l.cfg.AzureOpenAIDeploymentName),
		}, nil)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: LLM API call failed: %w", attempt+1, err)

			continue
		}

		err = json.Unmarshal([]byte(*resp.Choices[0].Message.Content), &specs)
		if err == nil {
			break
		}

		lastErr = fmt.Errorf("attempt %d: failed to unmarshal agent specs: %w", attempt+1, err)
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed to unmarshal agent specs after %d attempts: %w", maxRetries, lastErr)
	}

	return []*types.AgentExtension{
		{
			Name:    PluginName,
			Version: PluginVersion,
			Specs:   specs,
		},
	}, nil
}

func concatenateFiles(dirPath string, ignorePaths []string, extensions []string) (string, error) {
	var result strings.Builder

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		// skip files in ignore list
		for _, ignorePath := range ignorePaths {
			if strings.Contains(path, ignorePath) {
				return nil
			}
		}

		if err != nil {
			return fmt.Errorf("failed to walk path %s: %w", path, err)
		}

		if info.IsDir() {
			return nil
		}

		hasValidExt := false

		for _, ext := range extensions {
			if strings.HasSuffix(info.Name(), ext) {
				hasValidExt = true

				break
			}
		}

		if !hasValidExt {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", path, err)
		}

		result.Write(content)
		result.WriteString("\n\n")

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to concatenate files: %w", err)
	}

	return result.String(), nil
}
