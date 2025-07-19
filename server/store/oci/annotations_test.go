// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"
	"time"

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
						SchemaVersion: "v1",
						CreatedAt:     "2023-01-01T00:00:00Z",
						Authors:       []string{"author1", "author2"},
					},
				},
			},
			contains: map[string]string{
				manifestDirObjectTypeKey: "record",
				ManifestKeyOASFVersion:   "v1",
				ManifestKeyName:          "test-agent",
				ManifestKeyVersion:       "1.0.0",
				ManifestKeyDescription:   "Test agent description",
				ManifestKeySchemaVersion: "v1",
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
				ManifestKeyName:                     "skill-agent",
				ManifestKeyVersion:                  "2.0.0",
				ManifestKeySkills:                   "processing,inference",
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
				ManifestKeyOASFVersion: "v3",
				ManifestKeyName:        "test-record-v2",
				ManifestKeyVersion:     "2.0.0",
				ManifestKeyDescription: "Test record v2 description",
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

func TestCreateDescriptorAnnotations(t *testing.T) {
	tests := []struct {
		name     string
		record   *corev1.Record
		contains map[string]string
	}{
		{
			name: "V1Alpha1 record",
			record: &corev1.Record{
				Data: &corev1.Record_V1{
					V1: &objectsv1.Agent{
						Name: "test-agent",
					},
				},
			},
			contains: map[string]string{
				DescriptorKeyBlobType:    "oasf-record",
				DescriptorKeyEncoding:    "json",
				DescriptorKeyCompression: "none",
				DescriptorKeySchema:      "oasf.v1.Agent",
				// DescriptorKeyContentCid is verified separately using record.GetCid()
				DescriptorKeySigned:       "false",
				DescriptorKeyStoreVersion: "v1",
			},
		},
		{
			name: "V1Alpha2 record",
			record: &corev1.Record{
				Data: &corev1.Record_V3{
					V3: &objectsv3.Record{
						Name: "test-record-v2",
					},
				},
			},
			contains: map[string]string{
				DescriptorKeyBlobType: "oasf-record",
				DescriptorKeyEncoding: "json",
				DescriptorKeySchema:   "oasf.v3.Record",
				DescriptorKeySigned:   "false",
			},
		},
		{
			name: "Record with signature",
			record: &corev1.Record{
				Data: &corev1.Record_V1{
					V1: &objectsv1.Agent{
						Name: "signed-agent",
						Signature: &objectsv1.Signature{
							Signature: "signature-data",
						},
					},
				},
			},
			contains: map[string]string{
				DescriptorKeySigned: "true",
			},
		},
		{
			name:   "Record with no data",
			record: &corev1.Record{},
			contains: map[string]string{
				DescriptorKeyBlobType: "oasf-record",
				DescriptorKeyEncoding: "json",
				DescriptorKeySchema:   "unknown",
				DescriptorKeySigned:   "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createDescriptorAnnotations(tt.record)

			// Check that all expected keys are present with correct values
			for key, expectedValue := range tt.contains {
				assert.Equal(t, expectedValue, result[key], "Key %s should have correct value", key)
			}

			// Verify that ContentCID matches the record's computed CID
			expectedCID := tt.record.GetCid()
			assert.Equal(t, expectedCID, result[DescriptorKeyContentCid], "ContentCID should match record's computed CID")

			// Verify timestamp format (should be valid RFC3339)
			if storedAt, exists := result[DescriptorKeyStoredAt]; exists {
				_, err := time.Parse(time.RFC3339, storedAt)
				assert.NoError(t, err, "StoredAt should be valid RFC3339 timestamp")
			}
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
				SchemaVersion: "v1", // fallback
				Annotations:   make(map[string]string),
			},
		},
		{
			name:        "Empty annotations",
			annotations: map[string]string{},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v1", // fallback
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
				SchemaVersion: "v1", // fallback
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
				SchemaVersion: "v1", // fallback
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
				SchemaVersion: "v1", // fallback
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
				SchemaVersion: "v1", // fallback
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

			assert.Equal(t, tt.expected.SchemaVersion, result.SchemaVersion)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)

			// Check all expected annotations
			for key, expectedValue := range tt.expected.Annotations {
				assert.Equal(t, expectedValue, result.Annotations[key], "Annotation key %s should have correct value", key)
			}

			// Ensure no unexpected annotations (allow for additional count fields)
			for key := range result.Annotations {
				if _, expected := tt.expected.Annotations[key]; !expected {
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
		assert.Equal(t, "v1", result[ManifestKeyOASFVersion])
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
	originalRecord := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name:          "roundtrip-agent",
				Version:       "1.0.0",
				Description:   "Test roundtrip conversion",
				SchemaVersion: "v1",
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
	assert.Equal(t, "v1", recordMeta.SchemaVersion)
	assert.Equal(t, "2023-01-01T00:00:00Z", recordMeta.CreatedAt)
	assert.Equal(t, "roundtrip-agent", recordMeta.Annotations[MetadataKeyName])
	assert.Equal(t, "1.0.0", recordMeta.Annotations[MetadataKeyVersion])
	assert.Equal(t, "Test roundtrip conversion", recordMeta.Annotations[MetadataKeyDescription])
	assert.Equal(t, "v1", recordMeta.Annotations[MetadataKeyOASFVersion])
	assert.Equal(t, "author1,author2", recordMeta.Annotations[MetadataKeyAuthors])
	assert.Equal(t, "2", recordMeta.Annotations[MetadataKeyAuthorsCount])
	assert.Equal(t, "processing", recordMeta.Annotations[MetadataKeySkills])
	assert.Equal(t, "1", recordMeta.Annotations[MetadataKeySkillsCount])
	assert.Equal(t, "value", recordMeta.Annotations["custom"])
}
