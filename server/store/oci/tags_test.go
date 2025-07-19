// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	corev1 "github.com/agntcy/dir/api/core/v1"
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	objectsv3 "github.com/agntcy/dir/api/objects/v3"
	"github.com/stretchr/testify/assert"
)

// Helper function to create string pointers
func TestNormalizeTagForOCI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Simple valid tag",
			input:    "my-agent",
			expected: "my-agent",
		},
		{
			name:     "Uppercase to lowercase",
			input:    "MyAgent",
			expected: "myagent",
		},
		{
			name:     "Spaces to hyphens",
			input:    "my agent name",
			expected: "my-agent-name",
		},
		{
			name:     "Path separators to dots",
			input:    "org/team/agent",
			expected: "org.team.agent",
		},
		{
			name:     "Invalid characters replaced",
			input:    "agent@v1.0!",
			expected: "agent_v1.0", // Trailing underscore removed by TrimRight
		},
		{
			name:     "Mixed invalid characters",
			input:    "My Agent/v1.0@company",
			expected: "my-agent.v1.0_company",
		},
		{
			name:     "Invalid first character",
			input:    ".agent",
			expected: "_agent",
		},
		{
			name:     "Invalid first character hyphen",
			input:    "-agent",
			expected: "_agent",
		},
		{
			name:     "Long tag truncation",
			input:    "very-long-agent-name-that-exceeds-normal-limits-and-should-be-truncated-to-reasonable-length-for-oci-registry-compatibility-and-more-characters-to-exceed-128-limit",
			expected: "very-long-agent-name-that-exceeds-normal-limits-and-should-be-truncated-to-reasonable-length-for-oci-registry-compatibility-and",
		},
		{
			name:     "Trailing separators removed",
			input:    "agent...",
			expected: "agent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeTagForOCI(tt.input)
			assert.Equal(t, tt.expected, result)

			// Additional validation for non-empty results
			if tt.expected != "" {
				// Should start with valid character
				first := result[0]
				assert.True(t, (first >= 'a' && first <= 'z') || (first >= '0' && first <= '9') || first == '_',
					"Tag should start with valid character: %c", first)

				// Should not end with separator
				if len(result) > 0 {
					last := result[len(result)-1]
					assert.True(t, last != '.' && last != '-' && last != '_',
						"Tag should not end with separator: %c", last)
				}
			}
		})
	}
}

func TestRemoveDuplicateTags(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Empty slice",
			input:    []string{},
			expected: nil, // removeDuplicateTags returns nil for empty input
		},
		{
			name:     "No duplicates",
			input:    []string{"tag1", "tag2", "tag3"},
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "Some duplicates",
			input:    []string{"tag1", "tag2", "tag1", "tag3", "tag2"},
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "All duplicates",
			input:    []string{"tag1", "tag1", "tag1"},
			expected: []string{"tag1"},
		},
		{
			name:     "Empty strings filtered out",
			input:    []string{"tag1", "", "tag2", "", "tag3"},
			expected: []string{"tag1", "tag2", "tag3"},
		},
		{
			name:     "Mixed empty and duplicates",
			input:    []string{"tag1", "", "tag1", "tag2", "", "tag2"},
			expected: []string{"tag1", "tag2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeDuplicateTags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateTagsFromMetadata(t *testing.T) {
	tests := []struct {
		name        string
		metadata    map[string]string
		cid         string
		strategy    TagStrategy
		expectedMin int      // Minimum expected tags
		contains    []string // Tags that should be present
		notContains []string // Tags that should not be present
	}{
		{
			name:        "Empty metadata",
			metadata:    map[string]string{},
			cid:         "QmTest123",
			strategy:    DefaultTagStrategy,
			expectedMin: 1, // Should at least have CID tag
			contains:    []string{"qmtest123"},
		},
		{
			name: "Name-based tags only",
			metadata: map[string]string{
				MetadataKeyName:    "my-agent",
				MetadataKeyVersion: "1.0.0",
			},
			cid: "QmTest123",
			strategy: TagStrategy{
				EnableNameTags:           true,
				EnableCapabilityTags:     false,
				EnableInfrastructureTags: false,
				EnableTeamTags:           false,
				EnableContentAddressable: true,
				MaxTagsPerRecord:         20,
			},
			expectedMin: 3, // CID + name + name:version + name:latest
			contains:    []string{"qmtest123", "my-agent", "my-agent_1.0.0", "my-agent_latest"},
		},
		{
			name: "Capability-based tags",
			metadata: map[string]string{
				MetadataKeySkills:         "processing,service", // Only ClassName is extracted
				MetadataKeyExtensionNames: "security,monitoring",
			},
			cid: "QmTest123",
			strategy: TagStrategy{
				EnableNameTags:           false,
				EnableCapabilityTags:     true,
				EnableInfrastructureTags: false,
				EnableTeamTags:           false,
				EnableContentAddressable: true,
				MaxTagsPerRecord:         20,
			},
			expectedMin: 5, // CID + 2 skills + 2 extensions
			contains:    []string{"skill.processing", "skill.service", "ext.security", "ext.monitoring"},
		},
		{
			name: "Team-based tags",
			metadata: map[string]string{
				MetadataKeyTeam:         "backend",
				MetadataKeyOrganization: "acme",
				MetadataKeyProject:      "ml-platform",
			},
			cid: "QmTest123",
			strategy: TagStrategy{
				EnableNameTags:           false,
				EnableCapabilityTags:     false,
				EnableInfrastructureTags: false,
				EnableTeamTags:           true,
				EnableContentAddressable: true,
				MaxTagsPerRecord:         20,
			},
			expectedMin: 4, // CID + team + org + project
			contains:    []string{"qmtest123", "team.backend", "org.acme", "project.ml-platform"},
		},
		{
			name: "All tags disabled except CID",
			metadata: map[string]string{
				MetadataKeyName:   "my-agent",
				MetadataKeySkills: "nlp",
			},
			cid: "QmTest123",
			strategy: TagStrategy{
				EnableNameTags:           false,
				EnableCapabilityTags:     false,
				EnableInfrastructureTags: false,
				EnableTeamTags:           false,
				EnableContentAddressable: true,
				MaxTagsPerRecord:         20,
			},
			expectedMin: 1,
			contains:    []string{"qmtest123"},
			notContains: []string{"my-agent", "skill.nlp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateTagsFromMetadata(tt.metadata, tt.cid, tt.strategy)

			assert.GreaterOrEqual(t, len(result), tt.expectedMin, "Should generate minimum expected tags")

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "Should contain tag: %s", expected)
			}

			for _, notExpected := range tt.notContains {
				assert.NotContains(t, result, notExpected, "Should not contain tag: %s", notExpected)
			}

			// Verify no duplicates
			assert.Equal(t, len(result), len(removeDuplicateTags(result)), "Result should not contain duplicates")
		})
	}
}

func TestExtractMetadataFromRecord(t *testing.T) {
	tests := []struct {
		name     string
		record   *corev1.Record
		expected map[string]string
	}{
		{
			name:     "Nil record",
			record:   nil,
			expected: map[string]string{},
		},
		{
			name: "V1Alpha1 record with basic data",
			record: &corev1.Record{
				Data: &corev1.Record_V1{
					V1: &objectsv1.Agent{
						Name:        "test-agent",
						Version:     "1.0.0",
						Description: "Test agent",
						Skills: []*objectsv1.Skill{
							{CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
							{CategoryName: stringPtr("translation"), ClassName: stringPtr("service")},
						},
						Locators: []*objectsv1.Locator{
							{Type: "docker"},
							{Type: "helm"},
						},
						Extensions: []*objectsv1.Extension{
							{Name: "security"},
							{Name: "monitoring"},
						},
						Annotations: map[string]string{
							"team":         "backend",
							"organization": "acme",
						},
					},
				},
			},
			expected: map[string]string{
				MetadataKeyName:           "test-agent",
				MetadataKeyVersion:        "1.0.0",
				MetadataKeySkills:         "processing,service", // Only ClassName is extracted
				MetadataKeyLocatorTypes:   "docker,helm",
				MetadataKeyExtensionNames: "security,monitoring",
				MetadataKeyTeam:           "backend",
				MetadataKeyOrganization:   "acme",
			},
		},
		{
			name: "V1Alpha2 record with basic data",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "test-record",
						Version:     "2.0.0",
						Description: "Test record v2",
						Skills: []*objectsv3.Skill{
							{Name: "machine-learning"},
						},
						Locators: []*objectsv3.Locator{
							{Type: "kubernetes"},
						},
						Extensions: []*objectsv3.Extension{
							{Name: "logging"},
						},
						Annotations: map[string]string{
							"project": "ai-platform",
						},
					},
				},
			},
			expected: map[string]string{
				MetadataKeyName:           "test-record",
				MetadataKeyVersion:        "2.0.0",
				MetadataKeySkills:         "machine-learning",
				MetadataKeyLocatorTypes:   "kubernetes",
				MetadataKeyExtensionNames: "logging",
				MetadataKeyProject:        "ai-platform",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMetadataFromRecord(tt.record)

			for key, expectedValue := range tt.expected {
				assert.Equal(t, expectedValue, result[key], "Metadata key %s should have correct value", key)
			}

			// Verify no unexpected keys (allow for empty case)
			if len(tt.expected) > 0 {
				for key := range result {
					_, exists := tt.expected[key]
					assert.True(t, exists, "Unexpected metadata key: %s", key)
				}
			}
		})
	}
}

func TestGenerateDiscoveryTags(t *testing.T) {
	// Test the main public function
	agent := &objectsv1.Agent{
		Name:    "test-agent",
		Version: "1.0.0",
		Skills: []*objectsv1.Skill{
			{CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
		},
	}

	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: agent,
		},
	}

	tags := generateDiscoveryTags(record, DefaultTagStrategy)

	assert.NotEmpty(t, tags, "Should generate tags for valid record")

	// Get the actual CID for testing
	actualCID := record.GetCid()
	assert.NotEmpty(t, actualCID, "Record should have a valid CID")

	// Check that CID tag exists (normalized to lowercase)
	normalizedCID := normalizeTagForOCI(actualCID)
	assert.Contains(t, tags, normalizedCID, "Should contain normalized CID tag")
	assert.Contains(t, tags, "test-agent", "Should contain name tag")
	assert.Contains(t, tags, "skill.processing", "Should contain skill tag")

	// Test with nil record
	nilTags := generateDiscoveryTags(nil, DefaultTagStrategy)
	assert.Empty(t, nilTags, "Should return empty slice for nil record")
}

func TestReconstructTagsFromRecord(t *testing.T) {
	metadata := map[string]string{
		MetadataKeyName:   "test-agent",
		MetadataKeySkills: "processing,service", // Only ClassName is extracted
	}
	cid := "QmTestCID123"

	tags := reconstructTagsFromRecord(metadata, cid)

	assert.NotEmpty(t, tags, "Should generate tags from metadata")
	normalizedCID := normalizeTagForOCI(cid)
	assert.Contains(t, tags, normalizedCID, "Should contain normalized CID tag")
	assert.Contains(t, tags, "test-agent", "Should contain name tag")
}
