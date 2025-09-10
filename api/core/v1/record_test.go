// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package corev1

import (
	"testing"

	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/decoding/v1"
	oasfv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha0"
	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha1"
	"github.com/agntcy/oasf-sdk/core/converter"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRecord_GetCid(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		want    string
		wantErr bool
	}{
		{
			name: "v0.3.1 agent record",
			record: &Record{
				Data: toV0Proto(t, &oasfv1alpha0.Record{
					Name:          "test-agent",
					SchemaVersion: "v0.3.1",
					Description:   "A test agent",
				}),
			},
			wantErr: false,
		},
		{
			name: "v0.5.0 record",
			record: &Record{
				Data: toV1Proto(t, &oasfv1alpha1.Record{
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
			},
			wantErr: false,
		},
		{
			name:    "nil record",
			record:  nil,
			wantErr: true,
		},
		{
			name:    "empty record",
			record:  &Record{},
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
	// Create two identical v0.3.1 records.
	record1 := &Record{
		Data: toV1Proto(t, &oasfv1alpha1.Record{
			Name:          "test-agent",
			SchemaVersion: "v0.7.0",
			Description:   "A test agent",
		}),
	}

	record2 := &Record{
		Data: toV1Proto(t, &oasfv1alpha1.Record{
			Name:          "test-agent",
			SchemaVersion: "v0.7.0",
			Description:   "A test agent",
		}),
	}

	// Both records should have the same CID.
	cid1 := record1.GetCid()
	cid2 := record2.GetCid()

	assert.Equal(t, cid1, cid2, "Identical v0.3.1 records should have identical CIDs")
}

func TestRecord_GetCid_CrossVersion_Difference(t *testing.T) {
	// Create two different records
	record1 := &Record{
		Data: toV0Proto(t, &oasfv1alpha0.Record{
			Name:          "test-agent",
			SchemaVersion: "v0.3.1",
			Description:   "A test agent",
		}),
	}

	record2 := &Record{
		Data: toV1Proto(t, &oasfv1alpha1.Record{
			Name:          "test-agent",
			SchemaVersion: "v0.7.0",
			Description:   "A test agent",
		}),
	}

	// Both records should have the same CID.
	cid1 := record1.GetCid()
	cid2 := record2.GetCid()

	assert.NotEqual(t, cid1, cid2, "Different record versions should have different CIDs")
}

func TestRecord_MustGetCid(t *testing.T) {
	record := &Record{
		Data: toV1Proto(t, &oasfv1alpha1.Record{
			Name:          "test-agent",
			SchemaVersion: "v0.7.0",
			Description:   "A test agent",
		}),
	}

	// MustGetCid should not panic for valid record.
	assert.NotPanics(t, func() {
		cid := record.MustGetCid()
		assert.NotEmpty(t, cid)
	})

	// MustGetCid should panic for nil record.
	var nilRecord *Record

	assert.Panics(t, func() {
		nilRecord.MustGetCid()
	})

	// MustGetCid should panic for empty record (no OASF data).
	emptyRecord := &Record{}

	assert.Panics(t, func() {
		emptyRecord.MustGetCid()
	})
}

func TestRecord_Validate(t *testing.T) {
	tests := []struct {
		name      string
		record    *Record
		wantValid bool
	}{
		{
			name: "valid v0.7.0 record",
			record: &Record{
				Data: toV1Proto(t, &oasfv1alpha1.Record{
					Name:          "valid-agent-v2",
					SchemaVersion: "v0.7.0",
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
					Modules: []*oasfv1alpha1.Module{
						{
							Name: "test-extension",
						},
					},
				}),
			},
			wantValid: true,
		},
		{
			name: "invalid v0.7.0 record (missing required fields)",
			record: &Record{
				Data: toV1Proto(t, &oasfv1alpha1.Record{
					Name:          "invalid-agent-v2",
					SchemaVersion: "v0.5.0",
					Description:   "An invalid agent record in v0.5.0 format",
					Version:       "1.0.0",
				}),
			},
			wantValid: false,
		},
		{
			name:      "nil record",
			record:    nil,
			wantValid: false,
		},
		{
			name:      "empty record",
			record:    &Record{},
			wantValid: false,
		},
		{
			name: "record with invalid generic data",
			record: &Record{
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
			valid, errors, err := tt.record.Validate()
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
		record   *Record
		wantResp *decodingv1.DecodeRecordResponse
		wantFail bool
	}{
		{
			name: "valid v0.3.1 record",
			record: &Record{
				Data: toV0Proto(t, &oasfv1alpha0.Record{
					Name:          "valid-agent-v2",
					SchemaVersion: "v0.3.1",
					Description:   "A valid agent record",
					Version:       "1.0.0",
					CreatedAt:     "2024-01-01T00:00:00Z",
				}),
			},
			wantResp: &decodingv1.DecodeRecordResponse{
				Record: &decodingv1.DecodeRecordResponse_V1Alpha0{
					V1Alpha0: &oasfv1alpha0.Record{
						Name:          "valid-agent-v2",
						SchemaVersion: "v0.3.1",
						Description:   "A valid agent record",
						Version:       "1.0.0",
						CreatedAt:     "2024-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			name: "valid v0.7.0 record",
			record: &Record{
				Data: toV1Proto(t, &oasfv1alpha1.Record{
					Name:          "valid-agent-v2",
					SchemaVersion: "v0.7.0",
					Description:   "A valid agent record",
					Version:       "1.0.0",
					CreatedAt:     "2024-01-01T00:00:00Z",
				}),
			},
			wantResp: &decodingv1.DecodeRecordResponse{
				Record: &decodingv1.DecodeRecordResponse_V1Alpha1{
					V1Alpha1: &oasfv1alpha1.Record{
						Name:          "valid-agent-v2",
						SchemaVersion: "v0.7.0",
						Description:   "A valid agent record",
						Version:       "1.0.0",
						CreatedAt:     "2024-01-01T00:00:00Z",
					},
				},
			},
		},
		{
			name:     "nil record",
			record:   nil,
			wantFail: true,
		},
		{
			name:     "empty record",
			record:   &Record{},
			wantFail: true,
		},
		{
			name: "record with invalid generic data",
			record: &Record{
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

			if got == nil || got.Record == nil {
				t.Errorf("Decode() got nil record, want %v", tt.wantResp)

				return
			}

			if !assert.EqualValues(t, tt.wantResp, got) {
				t.Errorf("Decode() got %v, want %v", got, tt.wantResp)
			}
		})
	}
}

func toV0Proto(t *testing.T, recordV0 *oasfv1alpha0.Record) *structpb.Struct {
	t.Helper()

	res, err := converter.StructToProto(recordV0)
	if err != nil {
		t.Fatalf("Failed to convert record: %v", err)
	}

	return res
}

func toV1Proto(t *testing.T, recordV1 *oasfv1alpha1.Record) *structpb.Struct {
	t.Helper()

	res, err := converter.StructToProto(recordV1)
	if err != nil {
		t.Fatalf("Failed to convert record: %v", err)
	}

	return res
}
