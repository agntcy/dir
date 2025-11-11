// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mark3labs/mcphost/sdk"
)

var logger = logging.Logger("importer/enricher")

//go:embed enricher.prompt.md
var defaultPromptTemplate string

const (
	DebugMode                  = false
	DefaultConfigFile          = "importer/enricher/mcphost.json"
	DefaultConfidenceThreshold = 0.5
)

type Config struct {
	ConfigFile     string // Path to mcphost configuration file (e.g., mcphost.json)
	PromptTemplate string // Optional: path to custom prompt template file or inline prompt (empty = use default)
}

type MCPHostClient struct {
	host           *sdk.MCPHost
	promptTemplate string
}

// EnrichedField represents a single enriched field (skill or domain) with metadata.
type EnrichedField struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// EnrichmentResponse represents the structured JSON response from the LLM.
type EnrichmentResponse struct {
	Skills []EnrichedField `json:"skills"`
}

func NewMCPHost(ctx context.Context, config Config) (*MCPHostClient, error) {
	// Initialize MCP Host
	host, err := sdk.New(ctx, &sdk.Options{
		ConfigFile: config.ConfigFile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
	}

	// Load prompt template
	promptTemplate, err := loadPromptTemplate(config.PromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to load prompt template: %w", err)
	}

	if DebugMode {
		runGetSchemaToolsPrompt(ctx, host)
	}

	return &MCPHostClient{
		host:           host,
		promptTemplate: promptTemplate,
	}, nil
}

// loadPromptTemplate loads the prompt template from config or uses the default embedded template.
// If promptTemplateConfig is empty, uses the embedded default.
// If promptTemplateConfig looks like a file path (contains "/" or ends with ".md"), loads from file.
// Otherwise, treats it as an inline prompt template string.
func loadPromptTemplate(promptTemplateConfig string) (string, error) {
	// Use default embedded template if no custom template specified
	if promptTemplateConfig == "" {
		logger.Debug("Using default embedded prompt template")

		return defaultPromptTemplate, nil
	}

	// Check if it looks like a file path
	if strings.Contains(promptTemplateConfig, "/") || strings.HasSuffix(promptTemplateConfig, ".md") {
		logger.Debug("Loading prompt template from file", "path", promptTemplateConfig)

		data, err := os.ReadFile(promptTemplateConfig)
		if err != nil {
			return "", fmt.Errorf("failed to read prompt template file %s: %w", promptTemplateConfig, err)
		}

		return string(data), nil
	}

	// Treat as inline prompt template
	logger.Debug("Using inline prompt template from config")

	return promptTemplateConfig, nil
}

func (c *MCPHostClient) Enrich(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	// Marshal the record to JSON
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Run prompt for skills
	skillResponse, err := c.runPrompt(ctx, recordJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to run prompt: %w", err)
	}

	enrichedFields, err := c.parseResponse(skillResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skills: %w", err)
	}

	// Filter by confidence threshold and add to record
	for _, field := range enrichedFields {
		if field.Confidence >= DefaultConfidenceThreshold {
			record.Skills = append(record.Skills, &typesv1alpha1.Skill{
				Name: field.Name,
			})

			logger.Debug("Added skill", "name", field.Name, "confidence", field.Confidence, "reasoning", field.Reasoning)
		} else {
			logger.Debug("Skipped low-confidence skill", "name", field.Name, "confidence", field.Confidence, "threshold", DefaultConfidenceThreshold)
		}
	}

	enrichedRecordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enriched record: %w", err)
	}

	logger.Debug("Enriched record", "record", string(enrichedRecordJSON))

	return record, nil
}

func runGetSchemaToolsPrompt(ctx context.Context, host *sdk.MCPHost) {
	// Get 3 OASF skills
	resp, err := host.Prompt(ctx, "Call the tool 'dir-mcp-server__agntcy_oasf_get_schema_skills' and return 3 skill names)")
	if err != nil {
		logger.Error("failed to get 3 OASF skills", "error", err)
	}

	logger.Info("3 OASF skills", "skills", resp)

	// Get 3 sub-skills for the skill natural_language_processing
	resp, err = host.Prompt(ctx, "Call the tool 'dir-mcp-server__agntcy_oasf_get_schema_skills' and return 3 sub-skills for the skill natural_language_processing")
	if err != nil {
		logger.Error("failed to get 3 sub-skills for natural_language_processing", "error", err)
	}

	logger.Info("3 sub-skills for natural_language_processing", "sub-skills", resp)
}

func (c *MCPHostClient) runPrompt(ctx context.Context, recordJSON []byte) (string, error) {
	prompt := c.promptTemplate + string(recordJSON)

	var (
		response string
		err      error
	)

	if DebugMode {
		logger.Info("Original record", "record", string(recordJSON))

		// Send a prompt and get response with callbacks to see tool usage
		response, err = c.host.PromptWithCallbacks(
			ctx,
			prompt,
			func(name, args string) {
				logger.Info("Calling tool", "tool", name)
			},
			func(name, args, result string, isError bool) {
				if isError {
					logger.Error("Tool failed", "tool", name)
				} else {
					logger.Info("Tool completed", "tool", name)
				}
			},
			func(chunk string) {
			},
		)
		if err != nil {
			return "", fmt.Errorf("failed to send prompt: %w", err)
		}

		logger.Info("Response", "response", response)

		return response, nil
	}

	// No debug, just send the prompt and get the response
	response, err = c.host.Prompt(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	return response, nil
}

func (c *MCPHostClient) parseResponse(response string) ([]EnrichedField, error) {
	// Trim the entire response first to remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Try to parse as structured JSON first
	var enrichmentResp EnrichmentResponse

	err := json.Unmarshal([]byte(response), &enrichmentResp)
	if err == nil {
		// Successfully parsed as JSON
		fields := enrichmentResp.Skills

		// Validate and filter fields
		validFields := make([]EnrichedField, 0, len(fields))
		for _, field := range fields {
			// Basic validation: must contain exactly one forward slash
			if strings.Count(field.Name, "/") != 1 {
				logger.Warn("Skipping invalid skill format (must be skill/sub_skill)", "skill", field.Name)

				continue
			}

			// Validate confidence is in valid range
			if field.Confidence < 0.0 || field.Confidence > 1.0 {
				logger.Warn("Skipping skill with invalid confidence", "skill", field.Name, "confidence", field.Confidence)

				continue
			}

			validFields = append(validFields, field)
		}

		if len(validFields) == 0 {
			return nil, errors.New("no valid skills found in JSON response")
		}

		return validFields, nil
	}

	return nil, fmt.Errorf("failed to parse response: %w", err)
}
