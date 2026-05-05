// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"errors"
	"testing"

	oasfv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

// fakeValidator is a controllable corev1.Validator used to unit-test
// (*Record).ValidateWith without requiring network access to a live OASF
// schema server.
type fakeValidator struct {
	valid    bool
	errors   []string
	warnings []string
	err      error
}

func (f *fakeValidator) ValidateRecord(_ context.Context, _ *structpb.Struct) (bool, []string, []string, error) {
	return f.valid, f.errors, f.warnings, f.err
}

// Compile-time assertion that the test fake satisfies the public interface.
var _ corev1.Validator = (*fakeValidator)(nil)

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
	// Use the real OASF SDK validator against the live schema server. This test exercises
	// the full validation path including network I/O and is the integration counterpart to
	// the fake-based unit tests below.
	v, err := validator.New("https://schema.oasf.outshift.com")
	require.NoError(t, err, "failed to construct OASF validator")

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
			valid, errors, err := tt.record.ValidateWith(context.Background(), v)
			if err != nil {
				if tt.wantValid {
					t.Errorf("ValidateWith() unexpected error: %v", err)
				}

				return
			}

			if valid != tt.wantValid {
				t.Errorf("ValidateWith() got valid = %v, errors = %v, want %v", valid, errors, tt.wantValid)
			}

			if !valid && len(errors) == 0 {
				t.Errorf("ValidateWith() expected errors for invalid record, got none")
			}
		})
	}
}

// TestRecord_ValidateWith_NilValidator verifies the early-return behaviour when callers
// forget to construct/inject a validator. The method must not panic and must surface a
// useful diagnostic instead.
func TestRecord_ValidateWith_NilValidator(t *testing.T) {
	record := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "0.7.0",
	})

	valid, msgs, err := record.ValidateWith(context.Background(), nil)

	require.NoError(t, err)
	assert.False(t, valid)
	assert.Equal(t, []string{"validator is nil"}, msgs)
}

// TestRecord_ValidateWith_FakeValidator drives ValidateWith through a hermetic fake so
// we can assert the error/warning prefixing and pass-through behaviour without depending
// on the live OASF schema server.
func TestRecord_ValidateWith_FakeValidator(t *testing.T) {
	record := corev1.New(&oasfv1alpha1.Record{
		Name:          "test-agent",
		SchemaVersion: "0.7.0",
	})

	t.Run("valid record passes through", func(t *testing.T) {
		v := &fakeValidator{valid: true}

		ok, msgs, err := record.ValidateWith(context.Background(), v)

		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, msgs)
	})

	t.Run("errors and warnings are prefixed and combined", func(t *testing.T) {
		v := &fakeValidator{
			valid:    false,
			errors:   []string{"missing required field 'authors'"},
			warnings: []string{"deprecated field 'foo'"},
		}

		ok, msgs, err := record.ValidateWith(context.Background(), v)

		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, []string{
			"ERROR: missing required field 'authors'",
			"WARNING: deprecated field 'foo'",
		}, msgs)
	})

	t.Run("transport errors are wrapped", func(t *testing.T) {
		sentinel := errors.New("connection refused")
		v := &fakeValidator{err: sentinel}

		ok, _, err := record.ValidateWith(context.Background(), v)

		require.Error(t, err)
		assert.False(t, ok)
		assert.ErrorIs(t, err, sentinel)
	})
}

func TestRecord_ValidateWith_RecordSize(t *testing.T) {
	// Exercise the early-return paths in ValidateWith that never reach the validator.
	// A real fake is supplied so we'd notice if the short-circuit ever regressed and a
	// nil/empty record reached ValidateRecord.
	tests := []struct {
		name      string
		record    *corev1.Record
		wantValid bool
		wantMsg   string
	}{
		{
			name:      "nil record",
			record:    nil,
			wantValid: false,
			wantMsg:   "record is nil",
		},
		{
			name:      "record with nil data",
			record:    &corev1.Record{},
			wantValid: false,
			wantMsg:   "record is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, msgs, err := tt.record.ValidateWith(context.Background(), &fakeValidator{valid: true})

			require.NoError(t, err)
			assert.Equal(t, tt.wantValid, valid)
			assert.Equal(t, []string{tt.wantMsg}, msgs)
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

func TestRecord_GetName(t *testing.T) {
	tests := []struct {
		name   string
		record *corev1.Record
		want   string
	}{
		{
			name: "returns name from valid record",
			record: corev1.New(&oasfv1alpha1.Record{
				Name:          "my-agent",
				SchemaVersion: "0.7.0",
			}),
			want: "my-agent",
		},
		{
			name:   "returns empty for nil record",
			record: nil,
			want:   "",
		},
		{
			name:   "returns empty for empty record",
			record: &corev1.Record{},
			want:   "",
		},
		{
			name: "returns empty when name field is absent",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"description": structpb.NewStringValue("no name here"),
					},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.record.GetName())
		})
	}
}

func TestRecord_GetVersion(t *testing.T) {
	tests := []struct {
		name   string
		record *corev1.Record
		want   string
	}{
		{
			name: "returns version from valid record",
			record: corev1.New(&oasfv1alpha1.Record{
				Name:          "my-agent",
				SchemaVersion: "0.7.0",
				Version:       "2.1.0",
			}),
			want: "2.1.0",
		},
		{
			name:   "returns empty for nil record",
			record: nil,
			want:   "",
		},
		{
			name:   "returns empty for empty record",
			record: &corev1.Record{},
			want:   "",
		},
		{
			name: "returns empty when version field is absent",
			record: &corev1.Record{
				Data: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": structpb.NewStringValue("no-version"),
					},
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.record.GetVersion())
		})
	}
}
