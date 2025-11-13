// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	// Get available schema versions from the OASF SDK
	availableVersions, err := validator.GetAvailableSchemaVersions()
	if err != nil {
		return nil, GetSchemaSkillsOutput{
			ErrorMessage: fmt.Sprintf("Failed to get available schema versions: %v", err),
		}, nil
	}

	// Validate the version parameter
	if input.Version == "" {
		return nil, GetSchemaSkillsOutput{
			ErrorMessage:      "Version parameter is required. Available versions: " + strings.Join(availableVersions, ", "),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Check if the requested version is available
	versionValid := false
	for _, version := range availableVersions {
		if input.Version == version {
			versionValid = true
			break
		}
	}

	if !versionValid {
		return nil, GetSchemaSkillsOutput{
			ErrorMessage:      fmt.Sprintf("Invalid version '%s'. Available versions: %s", input.Version, strings.Join(availableVersions, ", ")),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Get skills content using the OASF SDK
	skillsJSON, err := validator.GetSchemaSkills(input.Version)
	if err != nil {
		return nil, GetSchemaSkillsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to get skills from OASF %s schema: %v", input.Version, err),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Parse skills JSON
	var skillsData map[string]interface{}
	if err := json.Unmarshal(skillsJSON, &skillsData); err != nil {
		return nil, GetSchemaSkillsOutput{
			Version:           input.Version,
			ErrorMessage:      fmt.Sprintf("Failed to parse skills data: %v", err),
			AvailableVersions: availableVersions,
		}, nil
	}

	// Parse all skills from the flat map structure
	// Each key is a skill short name, value contains the full name with hierarchy
	var allSkills []SkillItem
	for _, skillDef := range skillsData {
		defMap, ok := skillDef.(map[string]interface{})
		if !ok {
			continue
		}

		skill := parseSkillFromSchema(defMap)
		if skill.Name != "" {
			allSkills = append(allSkills, skill)
		}
	}

	// Filter based on parent_skill parameter
	var resultSkills []SkillItem
	if input.ParentSkill != "" {
		// Return skills that are children of the parent_skill
		// Children have names like "parent_skill/child_skill"
		prefix := input.ParentSkill + "/"
		for _, skill := range allSkills {
			if strings.HasPrefix(skill.Name, prefix) {
				// Check if this is a direct child (no further slashes after prefix)
				remainder := strings.TrimPrefix(skill.Name, prefix)
				if !strings.Contains(remainder, "/") {
					resultSkills = append(resultSkills, skill)
				}
			}
		}

		if len(resultSkills) == 0 {
			return nil, GetSchemaSkillsOutput{
				Version:           input.Version,
				ParentSkill:       input.ParentSkill,
				ErrorMessage:      fmt.Sprintf("Parent skill '%s' not found or has no children", input.ParentSkill),
				AvailableVersions: availableVersions,
			}, nil
		}
	} else {
		// Return only top-level parent categories
		// Extract unique parent categories from skill names (part before first "/")
		parentCategories := make(map[string]bool)
		for _, skill := range allSkills {
			if idx := strings.Index(skill.Name, "/"); idx > 0 {
				parentCategory := skill.Name[:idx]
				if !parentCategories[parentCategory] {
					parentCategories[parentCategory] = true
					// Create a skill item for the parent category
					resultSkills = append(resultSkills, SkillItem{
						Name: parentCategory,
					})
				}
			}
		}
	}

	// Return the skills
	return nil, GetSchemaSkillsOutput{
		Version:           input.Version,
		Skills:            resultSkills,
		ParentSkill:       input.ParentSkill,
		AvailableVersions: availableVersions,
	}, nil
}

// parseSkillFromSchema extracts skill information from the schema definition.
func parseSkillFromSchema(defMap map[string]interface{}) SkillItem {
	skill := SkillItem{}

	// Extract title for caption
	if title, ok := defMap["title"].(string); ok {
		skill.Caption = title
	}

	// Extract properties
	props, ok := defMap["properties"].(map[string]interface{})
	if !ok {
		return skill
	}

	// Extract name
	if nameField, ok := props["name"].(map[string]interface{}); ok {
		if constVal, ok := nameField["const"].(string); ok {
			skill.Name = constVal
		}
	}

	// Extract ID
	if idField, ok := props["id"].(map[string]interface{}); ok {
		if constVal, ok := idField["const"].(float64); ok {
			skill.ID = int(constVal)
		}
	}

	return skill
}
