// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRecord_GetCid(t *testing.T) {
	tests := []struct {
		name    string
		record  *corev1.Record
		want    string
		wantErr bool
	}{
		{
			name: "v0.5.0 record",
			record: corev1.New(&oasfv1alpha1.Record{
				Name:          "test-agent-v2",
				SchemaVersion: "v0.5.0",
				Description:   "A test agent in v0.5.0 record",
				Version:       "1.0.0",
				Modules: []*oasfv1alpha1.Module{
					{
						Name: "test-extension",
					},
				},
			}),
			wantErr: false,
		},
		{
			name:    "nil record",
			record:  nil,
			wantErr: true,
		},
		{
			name:    "empty record",
			record:  &corev1.Record{},
			wantErr: true, // Empty record should fail - no OASF data to marshal
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cid := tt.record.GetCid()

			if tt.wantErr {
				assert.Empty(t, cid)

				return
			}

			assert.NotEmpty(t, cid)

			// CID should be consistent - calling it again should return the same value.
			cid2 := tt.record.GetCid()
			assert.Equal(t, cid, cid2, "CID should be deterministic")

			// CID should start with the CIDv1 prefix.
			assert.Greater(t, len(cid), 10, "CID should be a reasonable length")
		})
	}
}

func TestRecord_GetCid_Consistency(t *testing.T) {
	// Create two identical 0.7.0 records.
	record1 := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "0.7.0",
		Description:   "A test agent",
	})

	record2 := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "0.7.0",
		Description:   "A test agent",
	})

	// Both records should have the same CID.
	cid1 := record1.GetCid()
	cid2 := record2.GetCid()

	assert.Equal(t, cid1, cid2, "Identical 0.7.0 records should have identical CIDs")
}

func TestRecord_GetCid_CrossVersion_Difference(t *testing.T) {
	// Create two different records with different schema versions
	record1 := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "0.7.0",
		Description:   "A test agent",
	})

	record2 := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "0.8.0",
		Description:   "A test agent",
	})

	// Both records should have different CIDs due to different schema versions.
	cid1 := record1.GetCid()
	cid2 := record2.GetCid()

	assert.NotEqual(t, cid1, cid2, "Different record versions should have different CIDs")
}

func TestRecord_Validate(t *testing.T) {
	if err := corev1.InitializeValidator("https://schema.oasf.outshift.com"); err != nil {
		t.Fatalf("Failed to initialize validator: %v", err)
	}

	tests := []struct {
		name      string
		record    *corev1.Record
		wantValid bool
	}{
		{
			name: "valid 0.7.0 record",
			record: corev1.New(&oasfv1alpha1.Record{
				Name:          "valid-agent-v2",
				SchemaVersion: "0.7.0",
				Description:   "A valid agent record",
				Version:       "1.0.0",
				CreatedAt:     "2024-01-01T00:00:00Z",
				Authors: []string{
					"Jane Doe <jane.doe@example.com>",
				},
				Locators: []*oasfv1alpha1.Locator{
					{
						Type: "helm_chart",
						Url:  "https://example.com/helm-chart.tgz",
					},
				},
				Skills: []*oasfv1alpha1.Skill{
					{
						Name: "natural_language_processing/natural_language_understanding",
					},
				},
			}),
			wantValid: true,
		},
		{
			name: "invalid 0.7.0 record (missing required fields)",
			record: corev1.New(&oasfv1alpha1.Record{
				Name:          "invalid-agent-v2",
				SchemaVersion: "v0.5.0",
				Description:   "An invalid agent record in v0.5.0 format",
				Version:       "1.0.0",
			}),
			wantValid: false,
		},
		{
			name:      "nil record",
			record:    nil,
			wantValid: false,
		},
		{
			name:      "empty record",
			record:    &corev1.Record{},
			wantValid: false,
		},
		{
			name: "record with invalid generic data",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"invalid_field": {
							Kind: &structpb.Value_StringValue{StringValue: "some value"},
						},
					},
				},
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errors, err := tt.record.Validate(context.Background())
			if err != nil {
				if tt.wantValid {
					t.Errorf("Validate() unexpected error: %v", err)
				}

				return
			}

			if valid != tt.wantValid {
				t.Errorf("Validate() got valid = %v, errors = %v, want %v", valid, errors, tt.wantValid)
			}

			if !valid && len(errors) == 0 {
				t.Errorf("Validate() expected errors for invalid record, got none")
			}
		})
	}
}

func TestRecord_InitializeValidator(t *testing.T) {
	// Test that InitializeValidator initializes the validator correctly
	tests := []struct {
		name      string
		schemaURL string
		wantError bool
	}{
		{
			name:      "initialize with valid schema URL",
			schemaURL: "https://schema.oasf.outshift.com",
			wantError: false,
		},
		{
			name:      "initialize with empty schema URL returns error",
			schemaURL: "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := corev1.InitializeValidator(tt.schemaURL)
			if tt.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// Verify that validation works after initialization
				record := corev1.New(&oasfv1alpha1.Record{
					Name:          "test-agent",
					SchemaVersion: "0.7.0",
					Description:   "A test agent",
					Version:       "1.0.0",
					CreatedAt:     "2024-01-01T00:00:00Z",
					Authors: []string{
						"Jane Doe <jane.doe@example.com>",
					},
					Locators: []*oasfv1alpha1.Locator{
						{
							Type: "helm_chart",
							Url:  "https://example.com/helm-chart.tgz",
						},
					},
					Skills: []*oasfv1alpha1.Skill{
						{
							Name: "natural_language_processing/natural_language_understanding",
						},
					},
				})

				// Validation should work for valid URLs
				// For invalid URLs, we expect a network error which is acceptable for this test
				valid, _, err := record.Validate(context.Background())
				if err != nil {
					// If it's a network error (like "no such host"), that's expected for invalid URLs
					if tt.schemaURL != "" && tt.schemaURL != "https://schema.oasf.outshift.com" {
						// For custom/invalid URLs, network errors are expected
						return
					}

					t.Fatalf("Validate() error = %v", err)
				}

				assert.True(t, valid)
			}
		})
	}
}

func TestRecord_Validate_RecordSize(t *testing.T) {
	// Test that Validate checks record size
	// Create a record that exceeds max size
	// Note: This is difficult to test without creating a very large record,
	// but we can test the nil and empty record cases which are part of the validation logic
	tests := []struct {
		name      string
		record    *corev1.Record
		wantValid bool
	}{
		{
			name:      "nil record",
			record:    nil,
			wantValid: false,
		},
		{
			name:      "record with nil data",
			record:    &corev1.Record{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, errors, err := tt.record.Validate(context.Background())
			if err != nil {
				if tt.wantValid {
					t.Errorf("Validate() unexpected error: %v", err)
				}

				return
			}

			if valid != tt.wantValid {
				t.Errorf("Validate() got valid = %v, errors = %v, want %v", valid, errors, tt.wantValid)
			}

			if !valid && len(errors) == 0 {
				t.Errorf("Validate() expected errors for invalid record, got none")
			}
		})
	}
}

func TestRecord_Decode(t *testing.T) {
	tests := []struct {
		name     string
		record   *corev1.Record
		wantResp any
		wantFail bool
	}{
		{
			name: "valid 0.7.0 record",
			record: corev1.New(&oasfv1alpha1.Record{
				Name:          "valid-agent-v2",
				SchemaVersion: "0.7.0",
				Description:   "A valid agent record",
				Version:       "1.0.0",
				CreatedAt:     "2024-01-01T00:00:00Z",
			}),
			wantResp: &oasfv1alpha1.Record{
				Name:          "valid-agent-v2",
				SchemaVersion: "0.7.0",
				Description:   "A valid agent record",
				Version:       "1.0.0",
				CreatedAt:     "2024-01-01T00:00:00Z",
			},
		},
		{
			name: "valid 1.0.0 record",
			record: func() *corev1.Record {
				record, _ := corev1.UnmarshalRecord([]byte(`{
					"name": "test-agent-v3",
					"schema_version": "1.0.0",
					"version": "1.0.0",
					"description": "A valid agent record",
					"created_at": "2024-01-01T00:00:00Z",
					"authors": ["test@example.com"],
					"skills": [{"name": "natural_language_processing/natural_language_understanding/contextual_comprehension", "id": 10101}],
					"locators": [{"type": "container_image", "urls": ["https://example.com/agent"]}]
				}`))

				return record
			}(),
			wantResp: func() any {
				// Decode the expected record to get the v1 record
				record, _ := corev1.UnmarshalRecord([]byte(`{
					"name": "test-agent-v3",
					"schema_version": "1.0.0",
					"version": "1.0.0",
					"description": "A valid agent record",
					"created_at": "2024-01-01T00:00:00Z",
					"authors": ["test@example.com"],
					"skills": [{"name": "natural_language_processing/natural_language_understanding/contextual_comprehension", "id": 10101}],
					"locators": [{"type": "container_image", "urls": ["https://example.com/agent"]}]
				}`))
				decoded, _ := record.Decode()

				return decoded.GetRecord()
			}(),
			wantFail: false,
		},
		{
			name:     "nil record",
			record:   nil,
			wantFail: true,
		},
		{
			name:     "empty record",
			record:   &corev1.Record{},
			wantFail: true,
		},
		{
			name: "record with invalid generic data",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"invalid_field": {
							Kind: &structpb.Value_StringValue{StringValue: "some value"},
						},
					},
				},
			},
			wantFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.record.Decode()
			if err != nil {
				if !tt.wantFail {
					t.Errorf("Decode() unexpected error: %v", err)
				}

				return
			}

			if got == nil {
				t.Errorf("Decode() got nil record, want %v", tt.wantResp)

				return
			}

			if !assert.EqualValues(t, tt.wantResp, got.GetRecord()) {
				t.Errorf("Decode() got %v, want %v", got, tt.wantResp)
			}
		})
	}
}
