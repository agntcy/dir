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
	DebugMode                  = false
	DefaultConfigFile          = "importer/enricher/mcphost.json"
	DefaultConfidenceThreshold = 0.5
	DefaultPromptTemplate      = `Select 1-3 skills for this agent record.

YOU MUST CALL THE TOOLS BELOW - DO NOT SKIP TOOL CALLS!
DO NOT INVENT SKILL NAMES - ONLY USE NAMES FROM TOOL RESPONSES!

MANDATORY STEPS - FOLLOW EXACTLY:

STEP 1: Call the first tool
   Tool: dir-mcp-server__agntcy_oasf_get_schema_skills
   Parameters: {"version": "0.7.0"}
   This returns ALL valid top-level skills with their "name" field.

STEP 2: Choose ONE top-level skill
   Pick ONE top-level skill from Step 1's tool response.
   Look at the "name" field of each skill.
   Choose the one that best matches the record below.

STEP 3: Call the second tool
   Tool: dir-mcp-server__agntcy_oasf_get_schema_skills
   Parameters: {"version": "0.7.0", "parent_skill": "the_name_from_step_2"}
   This returns ALL valid sub-skills with their "name" field.

STEP 4: Choose 1-3 sub-skills
   Pick 1-3 sub-skills from Step 3's tool response.
   Look ONLY at the "name" field of each sub-skill.
   COPY the exact "name" value - DO NOT change it!
   
   IMPORTANT: Tool response format:
   {"name": "actual_skill_name", "caption": "Human Label", "id": 123, ...}
   
   YOU MUST USE: The "name" field value EXACTLY as shown
   NEVER USE: The caption, description, or any other field
   NEVER INVENT: Names that don't appear in the tool response
   
   Real example from tool:
   {"name": "retrieval_of_information", "caption": "Indexing", "id": 601}
   ✓ CORRECT: "retrieval_augmented_generation/retrieval_of_information"
   ❌ WRONG: "retrieval_augmented_generation/indexing"

5. YOUR FINAL OUTPUT MUST BE VALID JSON ONLY - NO TEXT BEFORE OR AFTER:

{
  "skills": [
    {
      "name": "top_level_skill/sub_skill",
      "confidence": 0.95,
      "reasoning": "Brief explanation of why this skill matches"
    }
  ]
}

CRITICAL NAMING RULES - READ CAREFULLY:

1. You MUST call both tools (Step 1 and Step 3)
2. You MUST use ONLY the "name" field from the tool responses
3. Format MUST be: "top_level_skill/sub_skill" (exactly ONE slash)
4. DO NOT use top-level skills alone (like "tabular_text")
5. DO NOT invent skill names that sound plausible but weren't in the tools

EXAMPLES OF WRONG SKILL NAMES (DO NOT USE THESE):
❌ "tabular_text" - missing sub-skill, must be "tabular_text/something"
❌ "data_access/business_data_retrieval" - invented name, not in schema
❌ "api_management/api_server" - invented name, not in schema
❌ "machine_learning/statistical_analysis" - invented name, not in schema
❌ "data_analysis/data_processing" - invented name, not in schema
❌ "data_analysis/data_visualization" - invented name, not in schema
❌ "data_analysis/report_generation" - invented name, not in schema
❌ "retrieval_augmented_generation/indexing" - using caption, not name
❌ "retrieval_augmented_generation/document_retrieval" - doesn't exist

EXAMPLES OF CORRECT SKILL NAMES (THESE ARE REAL):
✓ "retrieval_augmented_generation/retrieval_of_information"
✓ "retrieval_augmented_generation/document_or_database_question_answering"
✓ "retrieval_augmented_generation/generation_of_any"
✓ "natural_language_processing/ethical_interaction"
✓ "natural_language_processing/information_retrieval_synthesis"
✓ "analytical_skills/mathematical_reasoning"
✓ "tabular_text/tabular_classification"
✓ "tabular_text/tabular_regression"

IF A SKILL NAME YOU WANT TO USE IS NOT IN THE TOOL RESPONSE, DON'T USE IT!
ONLY USE NAMES THAT APPEAR IN THE "name" FIELD OF THE TOOL RESPONSE!

CRITICAL OUTPUT RULES:
✓ Output ONLY valid JSON (no markdown code blocks, no explanations)
✓ Use exact skill names from tools (case-sensitive)
✓ Each name MUST be "top_level_skill/sub_skill" format with exactly one slash (/)
✓ Confidence must be a number between 0.0 and 1.0
✓ Reasoning should be 1-2 sentences explaining the match
✓ Include 1-3 skills in the array

❌ DO NOT wrap JSON in markdown code blocks (no triple backticks)
❌ DO NOT add text before or after the JSON
❌ DO NOT write: "Here is the JSON..." or "Based on..."
❌ DO NOT use skill names that were not in the tool response

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

FINAL REMINDER BEFORE YOU START:
1. Call tool #1 to get top-level skills
2. Pick ONE top-level skill from tool #1 response
3. Call tool #2 with that top-level skill to get sub-skills
4. Pick 1-3 sub-skills from tool #2 response
5. Use EXACT "name" field values from tool #2
6. Format as "top_level/sub_skill"
7. Output ONLY the JSON shown above

DO NOT make up skill names! Only use names from the tool responses!

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

	if DebugMode {
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
	prompt := DefaultPromptTemplate + string(recordJSON)

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
			return nil, fmt.Errorf("no valid skills found in JSON response")
		}

		return validFields, nil
	}

	return nil, fmt.Errorf("failed to parse response: %w", err)
}
