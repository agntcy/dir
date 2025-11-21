// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Intentional duplication with domains test for separate domain/skill testing
package tools

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSchemaSkills(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := []struct {
		name          string
		input         GetSchemaSkillsInput
		expectError   bool
		expectSkills  bool
		checkCallback func(t *testing.T, output GetSchemaSkillsOutput)
	}{
		{
			name: "Get top-level skills for version 0.7.0",
			input: GetSchemaSkillsInput{
				Version: "0.7.0",
			},
			expectError:  false,
			expectSkills: true,
			checkCallback: func(t *testing.T, output GetSchemaSkillsOutput) {
				t.Helper()

				assert.Equal(t, "0.7.0", output.Version)
				assert.Empty(t, output.ErrorMessage)
				assert.NotEmpty(t, output.Skills)

				// Check that top-level skills have expected fields
				for _, skill := range output.Skills {
					assert.NotEmpty(t, skill.Name, "Each skill should have a name")
				}
			},
		},
		{
			name: "Get sub-skills for a parent skill",
			input: GetSchemaSkillsInput{
				Version:     "0.7.0",
				ParentSkill: "retrieval_augmented_generation",
			},
			expectError:  false,
			expectSkills: true,
			checkCallback: func(t *testing.T, output GetSchemaSkillsOutput) {
				t.Helper()

				assert.Equal(t, "0.7.0", output.Version)
				assert.Equal(t, "retrieval_augmented_generation", output.ParentSkill)
				assert.Empty(t, output.ErrorMessage)
				assert.NotEmpty(t, output.Skills)

				// All returned skills should be sub-skills
				for _, skill := range output.Skills {
					assert.NotEmpty(t, skill.Name, "Each sub-skill should have a name")
				}
			},
		},
		{
			name: "Invalid version",
			input: GetSchemaSkillsInput{
				Version: "99.99.99",
			},
			expectError:  false,
			expectSkills: false,
			checkCallback: func(t *testing.T, output GetSchemaSkillsOutput) {
				t.Helper()

				assert.NotEmpty(t, output.ErrorMessage)
				assert.Contains(t, output.ErrorMessage, "invalid version")
				assert.NotEmpty(t, output.AvailableVersions)
			},
		},
		{
			name: "Missing version parameter",
			input: GetSchemaSkillsInput{
				Version: "",
			},
			expectError:  false,
			expectSkills: false,
			checkCallback: func(t *testing.T, output GetSchemaSkillsOutput) {
				t.Helper()

				assert.NotEmpty(t, output.ErrorMessage)
				assert.Contains(t, output.ErrorMessage, "version parameter is required")
				assert.NotEmpty(t, output.AvailableVersions)
			},
		},
		{
			name: "Non-existent parent skill",
			input: GetSchemaSkillsInput{
				Version:     "0.7.0",
				ParentSkill: "non_existent_skill",
			},
			expectError:  false,
			expectSkills: false,
			checkCallback: func(t *testing.T, output GetSchemaSkillsOutput) {
				t.Helper()

				assert.NotEmpty(t, output.ErrorMessage)
				assert.Contains(t, output.ErrorMessage, "not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, output, err := GetSchemaSkills(ctx, nil, tt.input)

			if tt.expectError {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Nil(t, result) // Tool handlers typically return nil for result

			if tt.expectSkills {
				assert.NotEmpty(t, output.Skills, "Expected to receive skills")
			}

			if tt.checkCallback != nil {
				tt.checkCallback(t, output)
			}
		})
	}
}

func TestParseItemFromSchemaForSkills(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		defMap   map[string]interface{}
		expected schemaClass
	}{
		{
			name: "Parse skill with name, caption (title), and ID",
			defMap: map[string]interface{}{
				"title": "Test Skill Caption",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"const": "test_skill",
					},
					"id": map[string]interface{}{
						"const": float64(123),
					},
				},
			},
			expected: schemaClass{
				Name:    "test_skill",
				Caption: "Test Skill Caption",
				ID:      123,
			},
		},
		{
			name: "Parse skill with missing fields",
			defMap: map[string]interface{}{
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"const": "minimal_skill",
					},
				},
			},
			expected: schemaClass{
				Name: "minimal_skill",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := parseItemFromSchema(tt.defMap)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Caption, result.Caption)
			assert.Equal(t, tt.expected.ID, result.ID)
		})
	}
}
