// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mark3labs/mcphost/sdk"
)

var logger = logging.Logger("importer/enricher")

const (
	DebugEnabled               = true
	DefaultConfigFile          = "importer/enricher/mcphost.json"
	DefaultConfidenceThreshold = 0.5
	DefaultPromptTemplate      = `Select 1-3 skills for this agent record.

MANDATORY STEPS - FOLLOW EXACTLY:

1. Call tool: dir-mcp-server__agntcy_oasf_get_schema_skills
   Parameters: {"version": "0.7.0"}
   This returns ALL valid top-level skills.

2. Pick ONE top-level skill from the tool response that best matches the record below.

3. Call tool AGAIN: dir-mcp-server__agntcy_oasf_get_schema_skills
   Parameters: {"version": "0.7.0", "parent_skill": "your_chosen_skill"}
   This returns ALL valid sub-skills for that skill.

4. Pick 1-3 sub-skills from the second tool response.

5. YOUR FINAL OUTPUT MUST BE VALID JSON ONLY - NO TEXT BEFORE OR AFTER:

{
  "skills": [
    {
      "name": "skill/sub_skill",
      "confidence": 0.95,
      "reasoning": "Brief explanation of why this skill matches"
    }
  ]
}

CRITICAL OUTPUT RULES:
✓ Output ONLY valid JSON (no markdown code blocks, no explanations)
✓ Use exact skill names from tools (case-sensitive)
✓ Each name MUST be "skill/sub_skill" format with exactly one slash (/)
✓ Confidence must be a number between 0.0 and 1.0
✓ Reasoning should be 1-2 sentences explaining the match
✓ Include 1-3 skills in the array

❌ DO NOT wrap JSON in markdown code blocks (no triple backticks)
❌ DO NOT add text before or after the JSON
❌ DO NOT write: "Here is the JSON..." or "Based on..."

Example of CORRECT output (copy this structure exactly):
{
  "skills": [
    {
      "name": "audio/speech_recognition",
      "confidence": 0.95,
      "reasoning": "Agent processes spoken audio input into text"
    },
    {
      "name": "audio/audio_generation",
      "confidence": 0.85,
      "reasoning": "Agent can generate audio output from text"
    }
  ]
}

Agent record to analyze:
`
)

type Config struct {
	ConfigFile string `json:"config_file"`
}

type MCPHostClient struct {
	host *sdk.MCPHost
}

// EnrichedField represents a single enriched field (skill or domain) with metadata
type EnrichedField struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// EnrichmentResponse represents the structured JSON response from the LLM
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

	if DebugEnabled {
		runGetSchemaToolsPrompt(ctx, host)
	}

	return &MCPHostClient{host: host}, nil
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

			if DebugEnabled {
				logger.Info("Added skill", "name", field.Name, "confidence", field.Confidence, "reasoning", field.Reasoning)
			}
		} else {
			logger.Warn("Skipped low-confidence skill", "name", field.Name, "confidence", field.Confidence, "threshold", DefaultConfidenceThreshold)
		}
	}


	if DebugEnabled {
		enrichedRecordJSON, err := json.Marshal(record)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal enriched record: %w", err)
		}

		logger.Info("Enriched record", "record", string(enrichedRecordJSON))
	}

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
	prompt := DefaultPromptTemplate + string(recordJSON)

	var (
		response string
		err      error
	)

	if DebugEnabled {
		logger.Info("Prompt", "prompt", prompt)

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
			return nil, fmt.Errorf("no valid skills found in JSON response")
		}

		return validFields, nil
	}

	return nil, fmt.Errorf("failed to parse response: %w", err)
}
