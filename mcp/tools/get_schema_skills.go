// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Intentional duplication with domains file for separate domain/skill handling
package tools

import (
	"context"
	"fmt"

	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GetSchemaSkillsInput represents the input for getting OASF schema skills.
type GetSchemaSkillsInput struct {
	Version     string `json:"version"                jsonschema:"OASF schema version to retrieve skills from (e.g., 0.7.0, 0.8.0)"`
	ParentSkill string `json:"parent_skill,omitempty" jsonschema:"Optional parent skill name to filter sub-skills (e.g., 'retrieval_augmented_generation')"`
}

// SkillItem represents a skill in the OASF schema.
type SkillItem struct {
	Name    string `json:"name"`
	Caption string `json:"caption,omitempty"`
	ID      int    `json:"id,omitempty"`
}

// GetSchemaSkillsOutput represents the output after getting OASF schema skills.
type GetSchemaSkillsOutput struct {
	Version           string      `json:"version"                      jsonschema:"The requested OASF schema version"`
	Skills            []SkillItem `json:"skills"                       jsonschema:"List of skills (top-level or filtered by parent)"`
	ParentSkill       string      `json:"parent_skill,omitempty"       jsonschema:"The parent skill filter if specified"`
	ErrorMessage      string      `json:"error_message,omitempty"      jsonschema:"Error message if skill retrieval failed"`
	AvailableVersions []string    `json:"available_versions,omitempty" jsonschema:"List of available OASF schema versions"`
}

// GetSchemaSkills retrieves skills from the OASF schema for the specified version.
// If parent_skill is provided, returns only sub-skills under that parent.
// Otherwise, returns all top-level skills.
func GetSchemaSkills(_ context.Context, _ *mcp.CallToolRequest, input GetSchemaSkillsInput) (
	*mcp.CallToolResult,
	GetSchemaSkillsOutput,
	error,
) {
	availableVersions, err := validateVersion(input.Version)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaSkillsOutput{
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	skillsJSON, err := validator.GetSchemaSkills(input.Version)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaSkillsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to get skills from OASF %s schema: %v", input.Version, err),
			AvailableVersions: availableVersions,
		}, nil
	}

	allSkills, err := parseSchemaData(skillsJSON, parseItemFromSchema)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaSkillsOutput{
			Version:           input.Version,
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	resultSkills, err := filterSkills(allSkills, input.ParentSkill)
	if err != nil {
		//nolint:nilerr // MCP tools communicate errors through output, not error return
		return nil, GetSchemaSkillsOutput{
			Version:           input.Version,
			ParentSkill:       input.ParentSkill,
			ErrorMessage:      err.Error(),
			AvailableVersions: availableVersions,
		}, nil
	}

	return nil, GetSchemaSkillsOutput{
		Version:           input.Version,
		Skills:            convertToSkillItems(resultSkills),
		ParentSkill:       input.ParentSkill,
		AvailableVersions: availableVersions,
	}, nil
}

// filterSkills filters skills based on parent parameter.
func filterSkills(allSkills []schemaClass, parent string) ([]schemaClass, error) {
	if parent != "" {
		return filterChildItems(allSkills, parent)
	}

	return extractTopLevelCategories(allSkills), nil
}

// convertToSkillItems converts generic schema items to SkillItem type.
func convertToSkillItems(items []schemaClass) []SkillItem {
	skills := make([]SkillItem, len(items))

	for i, item := range items {
		skills[i] = SkillItem(item)
	}

	return skills
}
