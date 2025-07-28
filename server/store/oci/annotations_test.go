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

func TestParseCommaSeparated(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Single value",
			input:    "value1",
			expected: []string{"value1"},
		},
		{
			name:     "Multiple values",
			input:    "value1,value2,value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "Values with spaces",
			input:    "value1, value2 , value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "Empty values filtered out",
			input:    "value1,,value2, ,value3",
			expected: []string{"value1", "value2", "value3"},
		},
		{
			name:     "Only commas and spaces",
			input:    ", , ,",
			expected: nil,
		},
		{
			name:     "Trailing and leading commas",
			input:    ",value1,value2,",
			expected: []string{"value1", "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCommaSeparated(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractManifestAnnotations(t *testing.T) {
	// NOTE: This test covers different OASF versions with varying skill name formats:
	// - V1 (objects.v1): Skills use "categoryName/className" hierarchical format
	// - V2 (objects.v2): Skills use simple name strings
	// - V3 (objects.v3): Skills use simple name strings
	tests := []struct {
		name     string
		record   *corev1.Record
		expected map[string]string
		contains map[string]string // Keys that should be present
	}{
		{
			name:   "Nil record",
			record: nil,
			expected: map[string]string{
				manifestDirObjectTypeKey: "record",
			},
		},
		{
			name: "V1Alpha1 basic record",
			record: &corev1.Record{
				Data: &corev1.Record_V1{
					V1: &objectsv1.Agent{
						Name:          "test-agent",
						Version:       "1.0.0",
						Description:   "Test agent description",
						SchemaVersion: "v0.3.1",
						CreatedAt:     "2023-01-01T00:00:00Z",
						Authors:       []string{"author1", "author2"},
					},
				},
			},
			contains: map[string]string{
				manifestDirObjectTypeKey: "record",
				ManifestKeyOASFVersion:   "v0.3.1",
				ManifestKeyName:          "test-agent",
				ManifestKeyVersion:       "1.0.0",
				ManifestKeyDescription:   "Test agent description",
				ManifestKeySchemaVersion: "v0.3.1",
				ManifestKeyCreatedAt:     "2023-01-01T00:00:00Z",
				ManifestKeyAuthors:       "author1,author2",
				ManifestKeySigned:        "false",
			},
		},
		{
			name: "V1Alpha1 with skills and extensions",
			record: &corev1.Record{
				Data: &corev1.Record_V1{
					V1: &objectsv1.Agent{
						Name:    "skill-agent",
						Version: "2.0.0",
						Skills: []*objectsv1.Skill{
							{CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
							{CategoryName: stringPtr("ml"), ClassName: stringPtr("inference")},
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
							"custom1": "value1",
							"custom2": "value2",
						},
					},
				},
			},
			contains: map[string]string{
				ManifestKeyName:    "skill-agent",
				ManifestKeyVersion: "2.0.0",
				// NOTE: V1 skills use "categoryName/className" format, unlike V2/V3 which use simple names
				ManifestKeySkills:                   "nlp/processing,ml/inference",
				ManifestKeyLocatorTypes:             "docker,helm",
				ManifestKeyExtensionNames:           "security,monitoring",
				ManifestKeyCustomPrefix + "custom1": "value1",
				ManifestKeyCustomPrefix + "custom2": "value2",
			},
		},
		{
			name: "V1Alpha2 basic record",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name:        "test-record-v2",
						Version:     "2.0.0",
						Description: "Test record v2 description",
						Skills: []*objectsv3.Skill{
							{Name: "nlp-skill"},
						},
						PreviousRecordCid: stringPtr("QmPreviousCID123"),
					},
				},
			},
			contains: map[string]string{
				ManifestKeyOASFVersion: "v0.5.0",
				ManifestKeyName:        "test-record-v2",
				ManifestKeyVersion:     "2.0.0",
				ManifestKeyDescription: "Test record v2 description",
				// NOTE: V3 skills use simple names, unlike V1 which uses "categoryName/className"
				ManifestKeySkills:      "nlp-skill",
				ManifestKeyPreviousCid: "QmPreviousCID123",
				ManifestKeySigned:      "false",
			},
		},
		{
			name: "Record with signature",
			record: &corev1.Record{
				Data: &corev1.Record_V1{
					V1: &objectsv1.Agent{
						Name:    "signed-agent",
						Version: "1.0.0",
						Signature: &objectsv1.Signature{
							Algorithm: "ed25519",
							SignedAt:  "2023-01-01T12:00:00Z",
							Signature: "signature-bytes",
						},
					},
				},
			},
			contains: map[string]string{
				ManifestKeyName:          "signed-agent",
				ManifestKeySigned:        "true",
				ManifestKeySignatureAlgo: "ed25519",
				ManifestKeySignedAt:      "2023-01-01T12:00:00Z",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractManifestAnnotations(tt.record)

			// Check that all expected keys are present with correct values
			for key, expectedValue := range tt.contains {
				assert.Equal(t, expectedValue, result[key], "Key %s should have correct value", key)
			}

			// Always should have the type key
			assert.Equal(t, "record", result[manifestDirObjectTypeKey])
		})
	}
}

func TestParseManifestAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    *corev1.RecordMeta
	}{
		{
			name:        "Nil annotations",
			annotations: nil,
			expected: &corev1.RecordMeta{
				SchemaVersion: "v0.3.1", // fallback
				Annotations:   make(map[string]string),
			},
		},
		{
			name:        "Empty annotations",
			annotations: map[string]string{},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v0.3.1", // fallback
				Annotations:   make(map[string]string),
			},
		},
		{
			name: "Basic record metadata",
			annotations: map[string]string{
				ManifestKeySchemaVersion: "v1alpha1",
				ManifestKeyCreatedAt:     "2023-01-01T00:00:00Z",
				ManifestKeyName:          "test-agent",
				ManifestKeyVersion:       "1.0.0",
				ManifestKeyDescription:   "Test description",
				ManifestKeyOASFVersion:   "v1alpha1",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v1alpha1",
				CreatedAt:     "2023-01-01T00:00:00Z",
				Annotations: map[string]string{
					MetadataKeyName:        "test-agent",
					MetadataKeyVersion:     "1.0.0",
					MetadataKeyDescription: "Test description",
					MetadataKeyOASFVersion: "v1alpha1",
				},
			},
		},
		{
			name: "Record with skills and counts",
			annotations: map[string]string{
				ManifestKeyName:    "skill-agent",
				ManifestKeySkills:  "nlp,ml,vision",
				ManifestKeyAuthors: "author1,author2",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v0.3.1", // fallback
				Annotations: map[string]string{
					MetadataKeyName:         "skill-agent",
					MetadataKeySkills:       "nlp,ml,vision",
					MetadataKeySkillsCount:  "3",
					MetadataKeyAuthors:      "author1,author2",
					MetadataKeyAuthorsCount: "2",
				},
			},
		},
		{
			name: "Record with security information",
			annotations: map[string]string{
				ManifestKeyName:          "secure-agent",
				ManifestKeySigned:        "true",
				ManifestKeySignatureAlgo: "ed25519",
				ManifestKeySignedAt:      "2023-01-01T12:00:00Z",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v0.3.1", // fallback
				Annotations: map[string]string{
					MetadataKeyName:          "secure-agent",
					MetadataKeySigned:        "true",
					MetadataKeySignatureAlgo: "ed25519",
					MetadataKeySignedAt:      "2023-01-01T12:00:00Z",
				},
			},
		},
		{
			name: "Record with custom annotations",
			annotations: map[string]string{
				ManifestKeyName:                     "custom-agent",
				ManifestKeyCustomPrefix + "custom1": "value1",
				ManifestKeyCustomPrefix + "custom2": "value2",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v0.3.1", // fallback
				Annotations: map[string]string{
					MetadataKeyName: "custom-agent",
					"custom1":       "value1",
					"custom2":       "value2",
				},
			},
		},
		{
			name: "Record with all metadata types",
			annotations: map[string]string{
				ManifestKeyName:           "full-agent",
				ManifestKeySkills:         "nlp,ml",
				ManifestKeyLocatorTypes:   "docker,helm,k8s",
				ManifestKeyExtensionNames: "security,monitoring",
				ManifestKeyPreviousCid:    "QmPrevious123",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v0.3.1", // fallback
				Annotations: map[string]string{
					MetadataKeyName:              "full-agent",
					MetadataKeySkills:            "nlp,ml",
					MetadataKeySkillsCount:       "2",
					MetadataKeyLocatorTypes:      "docker,helm,k8s",
					MetadataKeyLocatorTypesCount: "3",
					MetadataKeyExtensionNames:    "security,monitoring",
					MetadataKeyExtensionCount:    "2",
					MetadataKeyPreviousCid:       "QmPrevious123",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseManifestAnnotations(tt.annotations)

			assert.Equal(t, tt.expected.GetSchemaVersion(), result.GetSchemaVersion())
			assert.Equal(t, tt.expected.GetCreatedAt(), result.GetCreatedAt())

			// Check all expected annotations
			for key, expectedValue := range tt.expected.GetAnnotations() {
				assert.Equal(t, expectedValue, result.GetAnnotations()[key], "Annotation key %s should have correct value", key)
			}

			// Ensure no unexpected annotations (allow for additional count fields)
			for key := range result.GetAnnotations() {
				if _, expected := tt.expected.GetAnnotations()[key]; !expected {
					// Allow count fields that are auto-generated
					assert.True(t,
						key == MetadataKeySkillsCount ||
							key == MetadataKeyAuthorsCount ||
							key == MetadataKeyLocatorTypesCount ||
							key == MetadataKeyExtensionCount,
						"Unexpected annotation key: %s", key)
				}
			}
		})
	}
}

func TestExtractManifestAnnotations_EdgeCases(t *testing.T) {
	t.Run("Record with empty data", func(t *testing.T) {
		record := &corev1.Record{
			Data: &corev1.Record_V1{
				V1: &objectsv1.Agent{},
			},
		}

		result := extractManifestAnnotations(record)

		// Should still have basic annotations
		assert.Equal(t, "record", result[manifestDirObjectTypeKey])
		assert.Equal(t, "v0.3.1", result[ManifestKeyOASFVersion])
		assert.Equal(t, "false", result[ManifestKeySigned])
	})

	t.Run("Record with nil adapter data", func(t *testing.T) {
		record := &corev1.Record{} // No data field set

		result := extractManifestAnnotations(record)

		// Should return minimal annotations
		assert.Equal(t, "record", result[manifestDirObjectTypeKey])
		assert.Len(t, result, 1) // Only the type key
	})
}

func TestRoundTripConversion(t *testing.T) {
	// Test that we can extract manifest annotations and parse them back correctly
	// NOTE: This test uses V1 format where skills have "categoryName/className" structure
	originalRecord := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:          "roundtrip-agent",
				Version:       "1.0.0",
				Description:   "Test roundtrip conversion",
				SchemaVersion: "v0.3.1",
				CreatedAt:     "2023-01-01T00:00:00Z",
				Authors:       []string{"author1", "author2"},
				Skills: []*objectsv1.Skill{
					{CategoryName: stringPtr("nlp"), ClassName: stringPtr("processing")},
				},
				Annotations: map[string]string{
					"custom": "value",
				},
			},
		},
	}

	// Extract annotations
	manifestAnnotations := extractManifestAnnotations(originalRecord)

	// Parse them back
	recordMeta := parseManifestAnnotations(manifestAnnotations)

	// Verify round-trip conversion
	assert.Equal(t, "v0.3.1", recordMeta.GetSchemaVersion())
	assert.Equal(t, "2023-01-01T00:00:00Z", recordMeta.GetCreatedAt())
	assert.Equal(t, "roundtrip-agent", recordMeta.GetAnnotations()[MetadataKeyName])
	assert.Equal(t, "1.0.0", recordMeta.GetAnnotations()[MetadataKeyVersion])
	assert.Equal(t, "Test roundtrip conversion", recordMeta.GetAnnotations()[MetadataKeyDescription])
	assert.Equal(t, "v0.3.1", recordMeta.GetAnnotations()[MetadataKeyOASFVersion])
	assert.Equal(t, "author1,author2", recordMeta.GetAnnotations()[MetadataKeyAuthors])
	assert.Equal(t, "2", recordMeta.GetAnnotations()[MetadataKeyAuthorsCount])
	// NOTE: V1 skills return "categoryName/className" format, not just className
	assert.Equal(t, "nlp/processing", recordMeta.GetAnnotations()[MetadataKeySkills])
	assert.Equal(t, "1", recordMeta.GetAnnotations()[MetadataKeySkillsCount])
	assert.Equal(t, "value", recordMeta.GetAnnotations()["custom"])
}
