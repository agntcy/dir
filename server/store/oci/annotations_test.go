// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
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
	// NOTE: This test covers annotation extraction for different OASF versions
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
			name: "V070 basic record",
			record: corev1.New(&typesv1alpha1.Record{
				Name:          "test-agent",
				Version:       "1.0.0",
				Description:   "Test agent description",
				SchemaVersion: "0.7.0",
				CreatedAt:     "2023-01-01T00:00:00Z",
				Authors:       []string{"author1", "author2"},
			}),
			contains: map[string]string{
				manifestDirObjectTypeKey: "record",
				ManifestKeyOASFVersion:   "0.7.0",
				ManifestKeyName:          "test-agent",
				ManifestKeyVersion:       "1.0.0",
				ManifestKeyDescription:   "Test agent description",
				ManifestKeySchemaVersion: "0.7.0",
				ManifestKeyCreatedAt:     "2023-01-01T00:00:00Z",
				ManifestKeyAuthors:       "author1,author2",
			},
		},
		{
			name: "V070 with custom annotations",
			record: corev1.New(&typesv1alpha1.Record{
				Name:          "skill-agent",
				Version:       "2.0.0",
				SchemaVersion: "0.7.0",
				Annotations: map[string]string{
					"custom1": "value1",
					"custom2": "value2",
				},
			}),
			contains: map[string]string{
				ManifestKeyName:                     "skill-agent",
				ManifestKeyVersion:                  "2.0.0",
				ManifestKeyCustomPrefix + "custom1": "value1",
				ManifestKeyCustomPrefix + "custom2": "value2",
			},
		},
		{
			name: "V1 basic record",
			record: corev1.New(&typesv1alpha1.Record{
				Name:              "test-record-v2",
				Version:           "2.0.0",
				SchemaVersion:     "0.7.0",
				Description:       "Test record v2 description",
				PreviousRecordCid: stringPtr("QmPreviousCID123"),
			}),
			contains: map[string]string{
				ManifestKeyOASFVersion: "0.7.0",
				ManifestKeyName:        "test-record-v2",
				ManifestKeyVersion:     "2.0.0",
				ManifestKeyDescription: "Test record v2 description",
				ManifestKeyPreviousCid: "QmPreviousCID123",
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
				SchemaVersion: FallbackSchemaVersion,
				Annotations:   make(map[string]string),
			},
		},
		{
			name:        "Empty annotations",
			annotations: map[string]string{},
			expected: &corev1.RecordMeta{
				SchemaVersion: FallbackSchemaVersion,
				Annotations:   make(map[string]string),
			},
		},
		{
			name: "Basic record metadata",
			annotations: map[string]string{
				ManifestKeySchemaVersion: "v1",
				ManifestKeyCreatedAt:     "2023-01-01T00:00:00Z",
				ManifestKeyName:          "test-agent",
				ManifestKeyVersion:       "1.0.0",
				ManifestKeyDescription:   "Test description",
				ManifestKeyOASFVersion:   "v1",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: "v1",
				CreatedAt:     "2023-01-01T00:00:00Z",
				Annotations: map[string]string{
					MetadataKeyName:        "test-agent",
					MetadataKeyVersion:     "1.0.0",
					MetadataKeyDescription: "Test description",
					MetadataKeyOASFVersion: "v1",
				},
			},
		},
		{
			name: "Record with authors and counts",
			annotations: map[string]string{
				ManifestKeyName:    "author-agent",
				ManifestKeyAuthors: "author1,author2",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: FallbackSchemaVersion,
				Annotations: map[string]string{
					MetadataKeyName:         "author-agent",
					MetadataKeyAuthors:      "author1,author2",
					MetadataKeyAuthorsCount: "2",
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
				SchemaVersion: FallbackSchemaVersion,
				Annotations: map[string]string{
					MetadataKeyName: "custom-agent",
					"custom1":       "value1",
					"custom2":       "value2",
				},
			},
		},
		{
			name: "Record with versioning metadata",
			annotations: map[string]string{
				ManifestKeyName:        "versioned-agent",
				ManifestKeyPreviousCid: "QmPrevious123",
			},
			expected: &corev1.RecordMeta{
				SchemaVersion: FallbackSchemaVersion,
				Annotations: map[string]string{
					MetadataKeyName:        "versioned-agent",
					MetadataKeyPreviousCid: "QmPrevious123",
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
					assert.Equal(t, MetadataKeyAuthorsCount, key, "Unexpected annotation key: %s", key)
				}
			}
		})
	}
}

func TestExtractManifestAnnotations_EdgeCases(t *testing.T) {
	t.Run("Record with empty data", func(t *testing.T) {
		record := corev1.New(&typesv1alpha1.Record{
			SchemaVersion: "0.7.0",
		})

		result := extractManifestAnnotations(record)

		// Should still have basic annotations
		assert.Equal(t, "record", result[manifestDirObjectTypeKey])
		assert.Equal(t, "0.7.0", result[ManifestKeyOASFVersion])
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
	originalRecord := corev1.New(&typesv1alpha1.Record{
		Name:          "roundtrip-agent",
		Version:       "1.0.0",
		Description:   "Test roundtrip conversion",
		SchemaVersion: "0.7.0",
		CreatedAt:     "2023-01-01T00:00:00Z",
		Authors:       []string{"author1", "author2"},
		Annotations: map[string]string{
			"custom": "value",
		},
	})

	// Extract annotations
	manifestAnnotations := extractManifestAnnotations(originalRecord)

	// Parse them back
	recordMeta := parseManifestAnnotations(manifestAnnotations)

	// Verify round-trip conversion
	assert.Equal(t, "0.7.0", recordMeta.GetSchemaVersion())
	assert.Equal(t, "2023-01-01T00:00:00Z", recordMeta.GetCreatedAt())
	assert.Equal(t, "roundtrip-agent", recordMeta.GetAnnotations()[MetadataKeyName])
	assert.Equal(t, "1.0.0", recordMeta.GetAnnotations()[MetadataKeyVersion])
	assert.Equal(t, "Test roundtrip conversion", recordMeta.GetAnnotations()[MetadataKeyDescription])
	assert.Equal(t, "0.7.0", recordMeta.GetAnnotations()[MetadataKeyOASFVersion])
	assert.Equal(t, "author1,author2", recordMeta.GetAnnotations()[MetadataKeyAuthors])
	assert.Equal(t, "2", recordMeta.GetAnnotations()[MetadataKeyAuthorsCount])
	assert.Equal(t, "value", recordMeta.GetAnnotations()["custom"])
}
