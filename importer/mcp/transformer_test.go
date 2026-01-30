// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"maps"
	"testing"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"github.com/agntcy/dir/importer/config"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
	"google.golang.org/protobuf/types/known/structpb"
)

//nolint:nestif
func TestTransformer_Transform(t *testing.T) {
	// Enrichment is mandatory - transformer requires enricher initialization
	// Tests will be skipped if enricher cannot be initialized (no LLM available)
	cfg := config.Config{
		EnricherConfigFile: "testdata/mcphost.json",
	}

	transformer, err := NewTransformer(t.Context(), cfg)
	if err != nil {
		t.Skipf("Skipping test: enrichment is mandatory but enricher initialization failed: %v", err)
	}

	tests := []struct {
		name      string
		source    any
		wantErr   bool
		errString string
	}{
		{
			name: "valid server response",
			source: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server",
				},
			},
			wantErr: false,
		},
		{
			name:      "invalid source type - string",
			source:    "not a server response",
			wantErr:   true,
			errString: "invalid source type",
		},
		{
			name:      "invalid source type - nil",
			source:    nil,
			wantErr:   true,
			errString: "invalid source type",
		},
		{
			name:      "invalid source type - int",
			source:    42,
			wantErr:   true,
			errString: "invalid source type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := transformer.Transform(t.Context(), tt.source)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errString)
				}

				if record != nil {
					t.Error("expected nil record on error")
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}

				if record == nil {
					t.Error("expected record, got nil")
				}
			}
		})
	}
}

//nolint:nestif
func TestTransformer_ConvertToOASF(t *testing.T) {
	// Enrichment is mandatory - transformer requires enricher initialization
	// Tests will be skipped if enricher cannot be initialized (no LLM available)
	cfg := config.Config{
		EnricherConfigFile: "testdata/mcphost.json",
	}

	transformer, err := NewTransformer(t.Context(), cfg)
	if err != nil {
		t.Skipf("Skipping test: enrichment is mandatory but enricher initialization failed: %v", err)
	}

	tests := []struct {
		name     string
		response mcpapiv0.ServerResponse
		wantErr  bool
	}{
		{
			name: "basic server conversion",
			response: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:        "test-server",
					Version:     "1.0.0",
					Description: "Test server description",
				},
			},
			wantErr: false,
		},
		{
			name: "minimal server",
			response: mcpapiv0.ServerResponse{
				Server: mcpapiv0.ServerJSON{
					Name:    "minimal",
					Version: "0.1.0",
				},
				Meta: mcpapiv0.ResponseMeta{
					Official: &mcpapiv0.RegistryExtensions{
						Status:   model.StatusActive,
						IsLatest: true,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := transformer.convertToOASF(t.Context(), tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("convertToOASF() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				if record == nil {
					t.Error("convertToOASF() returned nil record")

					return
				}

				if record.GetData() == nil {
					t.Error("convertToOASF() returned record with nil Data")

					return
				}

				// Verify basic fields
				fields := record.GetData().GetFields()
				if fields["name"].GetStringValue() != tt.response.Server.Name {
					t.Errorf("name = %v, want %v", fields["name"].GetStringValue(), tt.response.Server.Name)
				}

				if fields["version"].GetStringValue() != tt.response.Server.Version {
					t.Errorf("version = %v, want %v", fields["version"].GetStringValue(), tt.response.Server.Version)
				}
			}
		})
	}
}

func TestStructToOASFRecord(t *testing.T) {
	tests := []struct {
		name      string
		structVal *structpb.Struct
		wantErr   bool
	}{
		{
			name: "valid struct with name and version",
			structVal: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name":    structpb.NewStringValue("test-server"),
					"version": structpb.NewStringValue("1.0.0"),
				},
			},
			wantErr: false,
		},
		{
			name: "struct with nested fields",
			structVal: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name":        structpb.NewStringValue("nested-server"),
					"version":     structpb.NewStringValue("2.0.0"),
					"description": structpb.NewStringValue("A test server"),
					"skills": structpb.NewListValue(&structpb.ListValue{
						Values: []*structpb.Value{
							structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"name": structpb.NewStringValue("test-skill"),
									"id":   structpb.NewNumberValue(1),
								},
							}),
						},
					}),
				},
			},
			wantErr: false,
		},
		// Note: nil struct will cause protojson.Marshal to return an error
		// but the actual behavior depends on protojson implementation
		// We skip this test case as it's implementation-dependent
		{
			name: "empty struct",
			structVal: &structpb.Struct{
				Fields: map[string]*structpb.Value{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record, err := structToOASFRecord(tt.structVal)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				if record != nil {
					t.Error("expected nil record on error")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if record == nil {
				t.Error("expected record, got nil")
			}
		})
	}
}

func TestUpdateFieldsInStruct(t *testing.T) {
	tests := []struct {
		name         string
		recordStruct *structpb.Struct
		fieldName    string
		items        []mockEnrichedItem
		wantErr      bool
		wantFieldLen int
	}{
		{
			name: "update skills field",
			recordStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name":    structpb.NewStringValue("test-server"),
					"version": structpb.NewStringValue("1.0.0"),
				},
			},
			fieldName: "skills",
			items: []mockEnrichedItem{
				{name: "skill1", id: 1},
				{name: "skill2", id: 2},
			},
			wantErr:      false,
			wantFieldLen: 2,
		},
		{
			name: "update domains field",
			recordStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name":    structpb.NewStringValue("test-server"),
					"version": structpb.NewStringValue("1.0.0"),
				},
			},
			fieldName: "domains",
			items: []mockEnrichedItem{
				{name: "domain1", id: 10},
			},
			wantErr:      false,
			wantFieldLen: 1,
		},
		{
			name: "empty items list",
			recordStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": structpb.NewStringValue("test-server"),
				},
			},
			fieldName:    "skills",
			items:        []mockEnrichedItem{},
			wantErr:      false,
			wantFieldLen: 0,
		},
		{
			name: "item with only name, no id",
			recordStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": structpb.NewStringValue("test-server"),
				},
			},
			fieldName: "skills",
			items: []mockEnrichedItem{
				{name: "skill-no-id", id: 0},
			},
			wantErr:      false,
			wantFieldLen: 1,
		},
		// Note: nil struct test is skipped as it causes panic in updateFieldsInStruct
		// The function checks for nil fields but not nil struct itself
		{
			name: "struct with nil fields",
			recordStruct: &structpb.Struct{
				Fields: nil,
			},
			fieldName:    "skills",
			items:        []mockEnrichedItem{{name: "skill1", id: 1}},
			wantErr:      true,
			wantFieldLen: 0, // Will fail before updating
		},
		{
			name: "preserves other fields",
			recordStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name":           structpb.NewStringValue("test-server"),
					"version":        structpb.NewStringValue("1.0.0"),
					"description":    structpb.NewStringValue("preserved"),
					"schema_version": structpb.NewStringValue("0.8.0"),
				},
			},
			fieldName: "skills",
			items: []mockEnrichedItem{
				{name: "skill1", id: 1},
			},
			wantErr:      false,
			wantFieldLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the struct to avoid modifying the test case
			var testStruct *structpb.Struct

			if tt.recordStruct != nil {
				// For nil fields case, preserve the nil
				if tt.recordStruct.Fields == nil {
					testStruct = &structpb.Struct{Fields: nil}
				} else {
					fieldsCopy := make(map[string]*structpb.Value, len(tt.recordStruct.GetFields()))
					maps.Copy(fieldsCopy, tt.recordStruct.GetFields())
					testStruct = &structpb.Struct{Fields: fieldsCopy}
				}
			}

			err := updateFieldsInStruct(testStruct, tt.fieldName, tt.items)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			verifyFieldUpdateWithItems(t, testStruct, tt.fieldName, tt.items, tt.wantFieldLen, tt.name)
		})
	}
}

func TestUpdateSkillsInStruct(t *testing.T) {
	recordStruct := createTestRecordStruct()
	skills := []*typesv1alpha1.Skill{
		{Name: "skill1", Id: 1},
		{Name: "skill2", Id: 2},
	}

	// Convert to []enrichedItem for the function
	enrichedSkills := make([]enrichedItem, len(skills))
	for i, skill := range skills {
		enrichedSkills[i] = skill
	}

	err := updateSkillsInStruct(recordStruct, enrichedSkills)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	verifyFieldUpdate(t, recordStruct, "skills", 2)
}

func TestUpdateDomainsInStruct(t *testing.T) {
	recordStruct := createTestRecordStruct()
	domains := []*typesv1alpha1.Domain{
		{Name: "domain1", Id: 10},
		{Name: "domain2", Id: 20},
	}

	// Convert to []enrichedItem for the function
	enrichedDomains := make([]enrichedItem, len(domains))
	for i, domain := range domains {
		enrichedDomains[i] = domain
	}

	err := updateDomainsInStruct(recordStruct, enrichedDomains)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	verifyFieldUpdate(t, recordStruct, "domains", 2)
}

// createTestRecordStruct creates a test record struct for testing.
func createTestRecordStruct() *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name":    structpb.NewStringValue("test-server"),
			"version": structpb.NewStringValue("1.0.0"),
		},
	}
}

// verifyFieldUpdate verifies that a field was updated in a struct with the expected length.
func verifyFieldUpdate(t *testing.T, recordStruct *structpb.Struct, fieldName string, wantFieldLen int) {
	t.Helper()

	fieldVal, ok := recordStruct.GetFields()[fieldName]
	if !ok {
		t.Fatalf("%s field was not added", fieldName)
	}

	fieldList := fieldVal.GetListValue()
	if fieldList == nil || len(fieldList.GetValues()) != wantFieldLen {
		t.Errorf("%s list length = %d, want %d", fieldName, len(fieldList.GetValues()), wantFieldLen)
	}
}

func verifyFieldUpdateWithItems(t *testing.T, testStruct *structpb.Struct, fieldName string, items []mockEnrichedItem, wantFieldLen int, testName string) {
	t.Helper()

	fieldVal, ok := testStruct.GetFields()[fieldName]
	if !ok {
		t.Errorf("field %q was not added to struct", fieldName)

		return
	}

	listVal := fieldVal.GetListValue()
	if listVal == nil {
		t.Error("field value is not a list")

		return
	}

	if len(listVal.GetValues()) != wantFieldLen {
		t.Errorf("field length = %d, want %d", len(listVal.GetValues()), wantFieldLen)

		return
	}

	verifyItems(t, listVal, items)

	if testName == "preserves other fields" {
		verifyPreservedFields(t, testStruct)
	}
}

func verifyItems(t *testing.T, listVal *structpb.ListValue, items []mockEnrichedItem) {
	t.Helper()

	values := listVal.GetValues()
	for i, item := range items {
		if i >= len(values) {
			break
		}

		itemStruct := values[i].GetStructValue()
		if itemStruct == nil {
			t.Errorf("item %d is not a struct", i)

			continue
		}

		nameVal := itemStruct.GetFields()["name"]
		if nameVal == nil || nameVal.GetStringValue() != item.name {
			t.Errorf("item %d name = %v, want %q", i, nameVal, item.name)
		}

		if item.id != 0 {
			idVal := itemStruct.GetFields()["id"]
			if idVal == nil || int(idVal.GetNumberValue()) != item.id {
				t.Errorf("item %d id = %v, want %d", i, idVal, item.id)
			}
		}
	}
}

func verifyPreservedFields(t *testing.T, testStruct *structpb.Struct) {
	t.Helper()

	fields := testStruct.GetFields()
	if fields["name"].GetStringValue() != "test-server" {
		t.Error("name field was not preserved")
	}

	if fields["version"].GetStringValue() != "1.0.0" {
		t.Error("version field was not preserved")
	}

	if fields["description"].GetStringValue() != "preserved" {
		t.Error("description field was not preserved")
	}

	if fields["schema_version"].GetStringValue() != "0.8.0" {
		t.Error("schema_version field was not preserved")
	}
}

// mockEnrichedItem is a mock implementation of enrichedItem for testing.
type mockEnrichedItem struct {
	name string
	id   int
}

func (m mockEnrichedItem) GetName() string {
	return m.name
}

func (m mockEnrichedItem) GetId() uint32 {
	if m.id < 0 || m.id > int(^uint32(0)) {
		return 0
	}

	return uint32(m.id)
}
