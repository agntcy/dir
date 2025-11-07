// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	"github.com/mark3labs/mcphost/sdk"
)

var logger = logging.Logger("importer/enricher")

const (
	DebugEnabled          = true
	DefaultConfigFile     = "importer/enricher/mcphost.json"
	DefaultPromptTemplate = `You are a #FIELD_NAME# selector.
	
	You are given a record and you need to select the most appropriate #FIELD_NAME#s that match the record's purpose.

	Call dir-mcp-server__agntcy_oasf_get_schema_#FIELD_NAME#s with version "0.7.0" to get the top level #FIELD_NAME#s.
	Based on the record's name and description, select the most appropriate top level #FIELD_NAME# that matches the record's purpose.

	Then call dir-mcp-server__agntcy_oasf_get_schema_#FIELD_NAME#s with version "0.7.0" to get the sub #FIELD_NAME#s of the selected top level #FIELD_NAME#.
	Based on the record's name and description, select the most appropriate sub #FIELD_NAME#s (1-3) that matches the record's purpose.

	Output ONLY the selected #FIELD_NAME# names as a comma separated list.
	EXAMPLE RESPONSE: top_level_#FIELD_NAME#1/sub_#FIELD_NAME#1,top_level_#FIELD_NAME#1/sub_#FIELD_NAME#2

	Here is the record:
	`
)

type Config struct {
	ConfigFile string `json:"config_file"`
}

type MCPHostClient struct {
	host *sdk.MCPHost
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

func (c *MCPHostClient) Enrich(ctx context.Context, record *corev1.Record) (*corev1.Record, error) {
	// Marshal the record to JSON
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Run prompt for skills
	skillResponse, err := c.runPrompt(ctx, "skill", recordJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to run prompt: %w", err)
	}

	// Decode the record to get the typed version
	decoded, err := record.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	// Get the V1Alpha1 record (assuming 0.7.0 schema)
	if !decoded.HasV1Alpha1() {
		return nil, errors.New("record is not V1Alpha1 format")
	}

	typedRecord := decoded.GetV1Alpha1()

	skills, err := c.parseResponse(skillResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse skills: %w", err)
	}

	for _, skill := range skills {
		typedRecord.Skills = append(typedRecord.Skills, &typesv1alpha1.Skill{
			Name: skill,
		})
	}

	// Re-encode the record to get the enriched record
	enrichedRecord := corev1.New(typedRecord)

	if DebugEnabled {
		enrichedRecordJSON, err := enrichedRecord.Marshal()
		if err != nil {
			return nil, fmt.Errorf("failed to marshal enriched record: %w", err)
		}

		logger.Info("Enriched record", "record", string(enrichedRecordJSON))
	}

	return enrichedRecord, nil
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

func (c *MCPHostClient) runPrompt(ctx context.Context, field string, recordJSON []byte) (string, error) {
	prompt := strings.ReplaceAll(DefaultPromptTemplate, "#FIELD_NAME#", field) + string(recordJSON)

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

		return response, nil
	}

	// No debug, just send the prompt and get the response
	response, err = c.host.Prompt(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	return response, nil
}

func (c *MCPHostClient) parseResponse(response string) ([]string, error) {
	items := strings.Split(response, ",")
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no valid skills found in response: %s", response)
	}

	return items, nil
}
