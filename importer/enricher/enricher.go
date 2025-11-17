// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mark3labs/mcphost/sdk"
)

var logger = logging.Logger("importer/enricher")

//go:embed enricher.skills.prompt.md
var defaultSkillsPromptTemplate string

//go:embed enricher.domains.prompt.md
var defaultDomainsPromptTemplate string

const (
	DebugMode                  = false
	DefaultConfigFile          = "importer/enricher/mcphost.json"
	DefaultConfidenceThreshold = 0.5
)

type Config struct {
	ConfigFile            string // Path to mcphost configuration file (e.g., mcphost.json)
	SkillsPromptTemplate  string // Optional: path to custom skills prompt template file or inline prompt (empty = use default)
	DomainsPromptTemplate string // Optional: path to custom domains prompt template file or inline prompt (empty = use default)
}

type MCPHostClient struct {
	host                  *sdk.MCPHost
	skillsPromptTemplate  string
	domainsPromptTemplate string
}

// EnrichedField represents a single enriched field (skill or domain) with metadata.
type EnrichedField struct {
	Name       string  `json:"name"`
	ID         uint32  `json:"id"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

// EnrichmentResponse represents the structured JSON response from the LLM.
// It can contain either skills or domains depending on the enrichment type.
type EnrichmentResponse struct {
	Skills  []EnrichedField `json:"skills,omitempty"`
	Domains []EnrichedField `json:"domains,omitempty"`
}

func NewMCPHost(ctx context.Context, config Config) (*MCPHostClient, error) {
	// Initialize MCP Host
	host, err := sdk.New(ctx, &sdk.Options{
		ConfigFile: config.ConfigFile,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MCPHost client: %w", err)
	}

	// Load prompt templates - use custom if provided, otherwise use defaults
	skillsPrompt, err := loadPromptTemplate(config.SkillsPromptTemplate, defaultSkillsPromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to load skills prompt template: %w", err)
	}

	domainsPrompt, err := loadPromptTemplate(config.DomainsPromptTemplate, defaultDomainsPromptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to load domains prompt template: %w", err)
	}

	if DebugMode {
		runGetSchemaToolsPrompt(ctx, host)
	}

	return &MCPHostClient{
		host:                  host,
		skillsPromptTemplate:  skillsPrompt,
		domainsPromptTemplate: domainsPrompt,
	}, nil
}

// loadPromptTemplate loads the prompt template from config or uses the provided default.
// If promptTemplateConfig is empty, uses the provided default template.
// If promptTemplateConfig looks like a file path (contains "/" or ends with ".md"), loads from file.
// Otherwise, treats it as an inline prompt template string.
func loadPromptTemplate(promptTemplateConfig, defaultTemplate string) (string, error) {
	// Use default embedded template if no custom template specified
	if promptTemplateConfig == "" {
		logger.Debug("Using default embedded prompt template")

		return defaultTemplate, nil
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

// fieldType represents the type of field being enriched (skills or domains).
type fieldType string

const (
	fieldTypeSkills  fieldType = "skills"
	fieldTypeDomains fieldType = "domains"
)

// EnrichWithSkills enriches the record with OASF skills using the LLM and MCP tools.
func (c *MCPHostClient) EnrichWithSkills(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	return c.enrichField(ctx, record, fieldTypeSkills, c.skillsPromptTemplate)
}

// EnrichWithDomains enriches the record with OASF domains using the LLM and MCP tools.
func (c *MCPHostClient) EnrichWithDomains(ctx context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	return c.enrichField(ctx, record, fieldTypeDomains, c.domainsPromptTemplate)
}

// enrichField is the generic enrichment method that handles both skills and domains.
func (c *MCPHostClient) enrichField(
	ctx context.Context,
	record *typesv1alpha1.Record,
	fType fieldType,
	promptTemplate string,
) (*typesv1alpha1.Record, error) {
	// Marshal the record to JSON
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Run prompt with the specified template
	response, err := c.runPrompt(ctx, promptTemplate, recordJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to run prompt for %s: %w", fType, err)
	}

	// Parse response to get enriched fields
	enrichedFields, err := c.parseResponse(response, fType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", fType, err)
	}

	// Filter by confidence threshold and add to record
	for _, field := range enrichedFields {
		if field.Confidence >= DefaultConfidenceThreshold {
			switch fType {
			case fieldTypeSkills:
				record.Skills = append(record.Skills, &typesv1alpha1.Skill{
					Name: field.Name,
					Id:   field.ID,
				})
			case fieldTypeDomains:
				record.Domains = append(record.Domains, &typesv1alpha1.Domain{
					Name: field.Name,
					Id:   field.ID,
				})
			}

			logger.Debug(fmt.Sprintf("Added %s", fType), "name", field.Name, "id", field.ID, "confidence", field.Confidence, "reasoning", field.Reasoning)
		} else {
			logger.Debug(fmt.Sprintf("Skipped low-confidence %s", fType), "name", field.Name, "confidence", field.Confidence, "threshold", DefaultConfidenceThreshold)
		}
	}

	enrichedRecordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal enriched record: %w", err)
	}

	logger.Debug(fmt.Sprintf("Enriched record with %s", fType), "record", string(enrichedRecordJSON))

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

func (c *MCPHostClient) runPrompt(ctx context.Context, promptTemplate string, recordJSON []byte) (string, error) {
	prompt := promptTemplate + string(recordJSON)

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

func (c *MCPHostClient) parseResponse(response string, fType fieldType) ([]EnrichedField, error) {
	// Trim the entire response first to remove leading/trailing whitespace
	response = strings.TrimSpace(response)

	// Try to parse as structured JSON first
	var enrichmentResp EnrichmentResponse

	err := json.Unmarshal([]byte(response), &enrichmentResp)
	if err == nil {
		// Get the appropriate field list based on type
		var fields []EnrichedField

		switch fType {
		case fieldTypeSkills:
			fields = enrichmentResp.Skills
		case fieldTypeDomains:
			fields = enrichmentResp.Domains
		default:
			return nil, fmt.Errorf("unknown field type: %s", fType)
		}

		// Validate and filter fields
		validFields := make([]EnrichedField, 0, len(fields))
		for _, field := range fields {
			// Basic validation: must contain exactly one forward slash
			if strings.Count(field.Name, "/") != 1 {
				logger.Warn(fmt.Sprintf("Skipping invalid %s format (must be parent/child)", fType), "name", field.Name)

				continue
			}

			// Validate ID is provided
			if field.ID == 0 {
				logger.Warn(fmt.Sprintf("Skipping %s without valid ID", fType), "name", field.Name)

				continue
			}

			// Validate confidence is in valid range
			if field.Confidence < 0.0 || field.Confidence > 1.0 {
				logger.Warn(fmt.Sprintf("Skipping %s with invalid confidence", fType), "name", field.Name, "confidence", field.Confidence)

				continue
			}

			validFields = append(validFields, field)
		}

		if len(validFields) == 0 {
			return nil, fmt.Errorf("no valid %s found in JSON response", fType)
		}

		return validFields, nil
	}

	return nil, fmt.Errorf("failed to parse response: %w", err)
}
