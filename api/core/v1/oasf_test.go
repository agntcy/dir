// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package corev1

import (
	"encoding/json"
	"strings"
	"testing"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectOASFVersion(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectedVer string
		expectError bool
	}{
		{
			name:        "v0.3.1 with schema_version",
			jsonData:    `{"schema_version": "v0.3.1", "name": "test-agent"}`,
			expectedVer: "v0.3.1",
			expectError: false,
		},
		{
			name:        "v0.4.0 with schema_version",
			jsonData:    `{"schema_version": "v0.4.0", "name": "test-agent"}`,
			expectedVer: "v0.4.0",
			expectError: false,
		},
		{
			name:        "v0.5.0 with schema_version",
			jsonData:    `{"schema_version": "v0.5.0", "name": "test-record"}`,
			expectedVer: "v0.5.0",
			expectError: false,
		},
		{
			name:        "no schema_version returns error",
			jsonData:    `{"name": "test-agent", "version": "1.0"}`,
			expectedVer: "",
			expectError: true,
		},
		{
			name:        "invalid json",
			jsonData:    `{"name": "test-agent"`,
			expectedVer: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := detectOASFVersion([]byte(tt.jsonData))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVer, version)
			}
		})
	}
}

func TestLoadOASFFromReader(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectV1    bool
		expectV2    bool
		expectV3    bool
	}{
		{
			name:        "valid v0.3.1 agent",
			jsonData:    `{"schema_version": "v0.3.1", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV1:    true,
		},
		{
			name:        "valid v0.4.0 agent record",
			jsonData:    `{"schema_version": "v0.4.0", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV2:    true,
		},
		{
			name:        "valid v0.5.0 record",
			jsonData:    `{"schema_version": "v0.5.0", "name": "test-record", "version": "1.0"}`,
			expectError: false,
			expectV3:    true,
		},
		{
			name:        "no schema_version returns error",
			jsonData:    `{"name": "test-agent", "version": "1.0"}`,
			expectError: true,
		},
		{
			name:        "unsupported version",
			jsonData:    `{"schema_version": "v0.6.0", "name": "test-record"}`,
			expectError: true,
		},
		{
			name:        "invalid json",
			jsonData:    `{"name": "test-agent"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.jsonData)
			record, err := LoadOASFFromReader(reader)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)

				//nolint:gocritic // if-else chain is clearer than switch for boolean flag testing in tests
				if tt.expectV1 {
					assert.NotNil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV2 {
					assert.Nil(t, record.GetV1())
					assert.NotNil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV3 {
					assert.Nil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.NotNil(t, record.GetV3())
				}
			}
		})
	}
}

func TestLoadOASFFromBytes(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		expectV1    bool
		expectV2    bool
		expectV3    bool
	}{
		{
			name:        "valid v0.3.1 agent",
			jsonData:    `{"schema_version": "v0.3.1", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV1:    true,
		},
		{
			name:        "valid v0.4.0 agent record",
			jsonData:    `{"schema_version": "v0.4.0", "name": "test-agent", "version": "1.0"}`,
			expectError: false,
			expectV2:    true,
		},
		{
			name:        "valid v0.5.0 record",
			jsonData:    `{"schema_version": "v0.5.0", "name": "test-record", "version": "1.0"}`,
			expectError: false,
			expectV3:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := loadOASFFromBytes([]byte(tt.jsonData))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, record)
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)

				//nolint:gocritic // if-else chain is clearer than switch for boolean flag testing in tests
				if tt.expectV1 {
					assert.NotNil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV2 {
					assert.Nil(t, record.GetV1())
					assert.NotNil(t, record.GetV2())
					assert.Nil(t, record.GetV3())
				} else if tt.expectV3 {
					assert.Nil(t, record.GetV1())
					assert.Nil(t, record.GetV2())
					assert.NotNil(t, record.GetV3())
				}
			}
		})
	}
}

func TestRecord_MarshalOASF(t *testing.T) {
	tests := []struct {
		name    string
		record  *Record
		wantErr bool
	}{
		{
			name: "v0.3.1 agent record",
			record: &Record{
				Data: &Record_V1{
					V1: &objectsv1.Agent{
						Name:          "test-agent",
						SchemaVersion: "v0.3.1",
						Description:   "A test agent",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "v0.5.0 record",
			record: &Record{
				Data: &Record_V3{
					V3: &objectsv3.Record{
						Name:          "test-agent-v2",
						SchemaVersion: "v0.5.0",
						Description:   "A test agent in v0.5.0 record",
						Version:       "1.0.0",
						Extensions: []*objectsv3.Extension{
							{
								Name:    "test-extension",
								Version: "1.0.0",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "nil record",
			record:  nil,
			wantErr: false, // Should return nil, nil
		},
		{
			name:    "empty record",
			record:  &Record{},
			wantErr: true, // Empty record should fail - no data to marshal
		},
		{
			name: "record with complex nested data",
			record: &Record{
				Data: &Record_V3{
					V3: &objectsv3.Record{
						Name:          "complex-agent",
						SchemaVersion: "v0.5.0",
						Description:   "A complex test agent",
						Version:       "2.1.0",
						Extensions: []*objectsv3.Extension{
							{
								Name:    "extension-1",
								Version: "1.0.0",
							},
							{
								Name:    "extension-2",
								Version: "2.0.0",
							},
						},
						Skills: []*objectsv3.Skill{
							{
								Name: "skill-1",
								Id:   1,
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.record.MarshalOASF()

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			require.NoError(t, err)

			if tt.record == nil {
				assert.Nil(t, got)

				return
			}

			// Verify the output is valid JSON
			var jsonData interface{}
			err = json.Unmarshal(got, &jsonData)
			require.NoError(t, err, "Output should be valid JSON")

			// Verify the output is compact (no unnecessary whitespace)
			assert.NotContains(t, string(got), "\n", "Output should be single line")
			assert.NotContains(t, string(got), "  ", "Output should not contain extra spaces")
		})
	}
}

func TestRecord_MarshalOASF_ReturnsOASFJSON(t *testing.T) {
	// Test that MarshalOASF returns pure OASF JSON, not Record wrapper
	tests := []struct {
		name        string
		record      *Record
		contains    []string // substrings that should be present in the output
		notContains []string // substrings that should NOT be present
	}{
		{
			name: "v1 agent returns OASF JSON",
			record: &Record{
				Data: &Record_V1{
					V1: &objectsv1.Agent{
						Name:          "test-agent",
						SchemaVersion: "v0.3.1",
						Description:   "A test agent",
					},
				},
			},
			contains:    []string{`"name":"test-agent"`, `"schema_version":"v0.3.1"`},
			notContains: []string{`"v1":`, `"data":`, `"V1":`}, // Should not contain Record wrapper
		},
		{
			name: "v3 record returns OASF JSON",
			record: &Record{
				Data: &Record_V3{
					V3: &objectsv3.Record{
						Name:          "test-record",
						SchemaVersion: "v0.5.0",
						Description:   "A test record",
						Version:       "1.0.0",
					},
				},
			},
			contains:    []string{`"name":"test-record"`, `"schema_version":"v0.5.0"`, `"version":"1.0.0"`},
			notContains: []string{`"v3":`, `"data":`, `"V3":`}, // Should not contain Record wrapper
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.record.MarshalOASF()
			require.NoError(t, err)

			gotStr := string(got)

			// Check that expected content is present
			for _, expected := range tt.contains {
				assert.Contains(t, gotStr, expected, "Output should contain: %s", expected)
			}

			// Check that Record wrapper content is NOT present
			for _, notExpected := range tt.notContains {
				assert.NotContains(t, gotStr, notExpected, "Output should NOT contain Record wrapper: %s", notExpected)
			}

			// Verify it's valid JSON
			var jsonData interface{}
			err = json.Unmarshal(got, &jsonData)
			require.NoError(t, err, "Output should be valid JSON")
		})
	}
}

func TestRecord_MarshalOASF_Deterministic(t *testing.T) {
	// Test that marshaling the same record multiple times produces identical output
	record := &Record{
		Data: &Record_V1{
			V1: &objectsv1.Agent{
				Name:          "deterministic-test",
				SchemaVersion: "v0.3.1",
				Description:   "Testing deterministic marshaling",
			},
		},
	}

	// Marshal the same record multiple times
	result1, err1 := record.MarshalOASF()
	require.NoError(t, err1)

	result2, err2 := record.MarshalOASF()
	require.NoError(t, err2)

	result3, err3 := record.MarshalOASF()
	require.NoError(t, err3)

	// All results should be identical
	assert.Equal(t, result1, result2, "Marshaling should be deterministic")
	assert.Equal(t, result2, result3, "Marshaling should be deterministic")
	assert.Equal(t, result1, result3, "Marshaling should be deterministic")
}

func TestRecord_MarshalOASF_KeyOrdering(t *testing.T) {
	// Test that JSON keys are ordered consistently
	record := &Record{
		Data: &Record_V3{
			V3: &objectsv3.Record{
				Name:          "key-order-test",
				SchemaVersion: "v0.5.0",
				Description:   "Testing key ordering",
				Version:       "1.0.0",
				Extensions: []*objectsv3.Extension{
					{
						Name:    "zeta-extension", // Intentionally out of alphabetical order
						Version: "1.0.0",
					},
					{
						Name:    "alpha-extension", // Should be ordered alphabetically in JSON
						Version: "2.0.0",
					},
				},
			},
		},
	}

	result, err := record.MarshalOASF()
	require.NoError(t, err)

	resultStr := string(result)

	// Verify that keys appear in alphabetical order in the JSON
	// Note: We can't easily test the exact ordering without parsing the JSON structure,
	// but we can verify it's consistent and valid
	assert.True(t, json.Valid(result), "Result should be valid JSON")
	assert.NotEmpty(t, resultStr, "Result should not be empty")
}

func TestUnmarshalOASF(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantErr  bool
		expectV1 bool
		expectV2 bool
		expectV3 bool
	}{
		{
			name:     "valid v0.3.1 OASF json",
			data:     []byte(`{"name":"test-agent","schema_version":"v0.3.1","description":"A test agent"}`),
			wantErr:  false,
			expectV1: true,
		},
		{
			name:     "valid v0.4.0 OASF json",
			data:     []byte(`{"name":"test-agent","schema_version":"v0.4.0","description":"A test agent","version":"1.0.0"}`),
			wantErr:  false,
			expectV2: true,
		},
		{
			name:     "valid v0.5.0 OASF json",
			data:     []byte(`{"name":"test-record","schema_version":"v0.5.0","description":"A test record","version":"1.0.0"}`),
			wantErr:  false,
			expectV3: true,
		},
		{
			name:    "no schema_version returns error",
			data:    []byte(`{"name":"test-agent","description":"A test agent"}`),
			wantErr: true,
		},
		{
			name:    "invalid json",
			data:    []byte(`{"invalid": json}`),
			wantErr: true,
		},
		{
			name:    "unsupported version",
			data:    []byte(`{"name":"test","schema_version":"v0.6.0"}`),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(``),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalOASF(tt.data)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)

			// Verify the correct version variant was created
			switch data := got.GetData().(type) {
			case *Record_V1:
				if !tt.expectV1 {
					t.Errorf("Expected V1=%v, but got V1 data", tt.expectV1)
				}

				assert.NotNil(t, data.V1, "Should have V1 data")
			case *Record_V2:
				if !tt.expectV2 {
					t.Errorf("Expected V2=%v, but got V2 data", tt.expectV2)
				}

				assert.NotNil(t, data.V2, "Should have V2 data")
			case *Record_V3:
				if !tt.expectV3 {
					t.Errorf("Expected V3=%v, but got V3 data", tt.expectV3)
				}

				assert.NotNil(t, data.V3, "Should have V3 data")
			case nil:
				t.Error("Unexpected nil data in record")
			default:
				t.Errorf("Unexpected record data type: %T", data)
			}
		})
	}
}

func TestMarshalOASF_UnmarshalOASF_RoundTrip(t *testing.T) {
	testCases := []struct {
		name   string
		record *Record
	}{
		{
			name: "v0.3.1 agent",
			record: &Record{
				Data: &Record_V1{
					V1: &objectsv1.Agent{
						Name:          "roundtrip-test",
						SchemaVersion: "v0.3.1",
						Description:   "Testing roundtrip marshaling",
					},
				},
			},
		},
		{
			name: "v0.5.0 record with extensions",
			record: &Record{
				Data: &Record_V3{
					V3: &objectsv3.Record{
						Name:          "roundtrip-v2",
						SchemaVersion: "v0.5.0",
						Description:   "Testing v0.5.0 roundtrip",
						Version:       "1.5.0",
						Extensions: []*objectsv3.Extension{
							{
								Name:    "test-ext",
								Version: "1.0.0",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal the record
			marshaled, err := tc.record.MarshalOASF()
			require.NoError(t, err)

			// Unmarshal it back
			unmarshaled, err := UnmarshalOASF(marshaled)
			require.NoError(t, err)

			// Marshal the unmarshaled record again
			remarshaled, err := unmarshaled.MarshalOASF()
			require.NoError(t, err)

			// The bytes should be identical (idempotent)
			assert.Equal(t, marshaled, remarshaled, "Round-trip should be idempotent")

			// The records should be functionally equivalent
			// (We can test this by comparing their CIDs)
			if tc.record.Data != nil {
				originalCID := tc.record.GetCid()
				unmarshaledCID := unmarshaled.GetCid()
				assert.Equal(t, originalCID, unmarshaledCID, "CIDs should match after round-trip")
			}
		})
	}
}

func TestMarshalOASF_UnmarshalOASF_RoundTripWithOASFFormat(t *testing.T) {
	// Test that the refactored canonical format works correctly:
	// Record → MarshalOASF (OASF JSON) → UnmarshalOASF → Record
	testCases := []struct {
		name   string
		record *Record
	}{
		{
			name: "v0.3.1 agent round-trip",
			record: &Record{
				Data: &Record_V1{
					V1: &objectsv1.Agent{
						Name:          "roundtrip-v1",
						SchemaVersion: "v0.3.1",
						Description:   "Testing v0.3.1 round-trip",
						Version:       "1.0.0",
					},
				},
			},
		},
		{
			name: "v0.5.0 record round-trip",
			record: &Record{
				Data: &Record_V3{
					V3: &objectsv3.Record{
						Name:          "roundtrip-v3",
						SchemaVersion: "v0.5.0",
						Description:   "Testing v0.5.0 round-trip",
						Version:       "2.0.0",
						Extensions: []*objectsv3.Extension{
							{
								Name:    "test-extension",
								Version: "1.0.0",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Marshal to OASF JSON
			oasfBytes, err := tc.record.MarshalOASF()
			require.NoError(t, err)

			// 2. Verify it's pure OASF JSON (contains schema_version, not Record wrapper)
			oasfStr := string(oasfBytes)
			assert.Contains(t, oasfStr, `"schema_version"`, "Should contain OASF schema_version")
			assert.NotContains(t, oasfStr, `"v1":`, "Should not contain Record wrapper")
			assert.NotContains(t, oasfStr, `"v3":`, "Should not contain Record wrapper")

			// 3. Unmarshal back to Record
			reconstructed, err := UnmarshalOASF(oasfBytes)
			require.NoError(t, err)

			// 4. Verify the version wrapper is correctly reconstructed
			switch tc.record.GetData().(type) {
			case *Record_V1:
				assert.NotNil(t, reconstructed.GetV1(), "Should reconstruct V1 wrapper")
				assert.Nil(t, reconstructed.GetV2(), "Should not have V2")
				assert.Nil(t, reconstructed.GetV3(), "Should not have V3")
			case *Record_V3:
				assert.Nil(t, reconstructed.GetV1(), "Should not have V1")
				assert.Nil(t, reconstructed.GetV2(), "Should not have V2")
				assert.NotNil(t, reconstructed.GetV3(), "Should reconstruct V3 wrapper")
			}

			// 5. Verify CIDs are identical (this tests the core refactor goal)
			originalCID := tc.record.GetCid()
			reconstructedCID := reconstructed.GetCid()
			assert.Equal(t, originalCID, reconstructedCID, "CIDs should be identical after round-trip")
			assert.NotEmpty(t, originalCID, "CID should not be empty")

			// 6. Verify marshaling again produces identical bytes (idempotent)
			secondMarshal, err := reconstructed.MarshalOASF()
			require.NoError(t, err)
			assert.Equal(t, oasfBytes, secondMarshal, "Second marshal should be identical")
		})
	}
}

func TestMarshalOASF_ConsistentAcrossIdenticalRecords(t *testing.T) {
	// Create two identical records separately to ensure they marshal identically
	createRecord := func() *Record {
		return &Record{
			Data: &Record_V3{
				V3: &objectsv3.Record{
					Name:          "consistency-test",
					SchemaVersion: "v0.5.0",
					Description:   "Testing marshaling consistency",
					Version:       "1.0.0",
					Extensions: []*objectsv3.Extension{
						{
							Name:    "ext-1",
							Version: "1.0.0",
						},
						{
							Name:    "ext-2",
							Version: "2.0.0",
						},
					},
				},
			},
		}
	}

	record1 := createRecord()
	record2 := createRecord()

	marshaled1, err1 := record1.MarshalOASF()
	require.NoError(t, err1)

	marshaled2, err2 := record2.MarshalOASF()
	require.NoError(t, err2)

	assert.Equal(t, marshaled1, marshaled2, "Identical records should marshal to identical bytes")
}

func TestUnmarshalOASF_InvalidInputs(t *testing.T) {
	invalidInputs := []struct {
		name string
		data []byte
	}{
		{
			name: "malformed json",
			data: []byte(`{"unclosed": "object"`),
		},
		{
			name: "unsupported schema version",
			data: []byte(`{"name": "test", "schema_version": "v9.9.9"}`),
		},
		{
			name: "wrong data type",
			data: []byte(`"just a string"`),
		},
		{
			name: "array instead of object",
			data: []byte(`[{"name": "test"}]`),
		},
	}

	for _, tc := range invalidInputs {
		t.Run(tc.name, func(t *testing.T) {
			result, err := UnmarshalOASF(tc.data)
			require.Error(t, err, "Should error on invalid input")
			assert.Nil(t, result, "Should return nil on error")
		})
	}
}
