// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/mark3labs/mcphost/sdk"
)

const (
	DefaultConfigFile  = "importer/enricher/mcphost.json"
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

	return &MCPHostClient{host: host}, nil
}

func (c *MCPHostClient) Enrich(ctx context.Context, record *corev1.Record) (*corev1.Record, error) {
	// Marshal the record to JSON
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Build skill prompt with record data
	skillPrompt := strings.Replace(DefaultPromptTemplate, "#FIELD_NAME#", "skill", -1) + string(recordJSON)

	// Send a prompt and get response with callbacks to see tool usage
	skillResponse, err := c.host.Prompt(ctx, skillPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	// Build domain prompt with record data
	domainPrompt := strings.Replace(DefaultPromptTemplate, "#FIELD_NAME#", "domain", -1) + string(recordJSON)

	// Send a prompt and get response with callbacks to see tool usage
	domainResponse, err := c.host.Prompt(ctx, domainPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to send prompt: %w", err)
	}

	// Decode the record to get the typed version
	decoded, err := record.Decode()
	if err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	// Get the V1Alpha1 record (assuming 0.7.0 schema)
	if !decoded.HasV1Alpha1() {
		return nil, fmt.Errorf("record is not V1Alpha1 format")
	}
	typedRecord := decoded.GetV1Alpha1()

	// Append the new skills and domains
	skills := strings.Split(skillResponse, ",")
	for _, skill := range skills {
		typedRecord.Skills = append(typedRecord.Skills, &typesv1alpha1.Skill{
			Name: skill,
		})
	}
	domains := strings.Split(domainResponse, ",")
	for _, domain := range domains {
		typedRecord.Domains = append(typedRecord.Domains, &typesv1alpha1.Domain{
			Name: domain,
		})
	}

	return corev1.New(typedRecord), nil
}
