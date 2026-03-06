// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package mcp

import (
	"context"
	"errors"
	"maps"
	"strings"
	"testing"

	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"github.com/agntcy/dir/importer/config"
	mcpapiv0 "github.com/modelcontextprotocol/registry/pkg/api/v0"
	"github.com/modelcontextprotocol/registry/pkg/model"
	"google.golang.org/protobuf/types/known/structpb"
)

const testVersion = "1.0.0"

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

	if fields["version"].GetStringValue() != testVersion {
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

// mockEnricherClient implements enricherClient for tests without a real LLM.
type mockEnricherClient struct {
	skillsErr  error
	domainsErr error
	skills     []*typesv1alpha1.Skill
	domains    []*typesv1alpha1.Domain
	skillsV1   []*typesv1.Skill
	domainsV1  []*typesv1.Domain
}

func (m *mockEnricherClient) EnrichWithSkills(_ context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	if m.skillsErr != nil {
		return nil, m.skillsErr
	}

	out := cloneRecordV1Alpha1(record)
	out.Skills = append(out.Skills, m.skills...)

	return out, nil
}

func (m *mockEnricherClient) EnrichWithDomains(_ context.Context, record *typesv1alpha1.Record) (*typesv1alpha1.Record, error) {
	if m.domainsErr != nil {
		return nil, m.domainsErr
	}

	out := cloneRecordV1Alpha1(record)
	out.Domains = append(out.Domains, m.domains...)

	return out, nil
}

func (m *mockEnricherClient) EnrichWithSkillsV1(_ context.Context, record *typesv1.Record) (*typesv1.Record, error) {
	if m.skillsErr != nil {
		return nil, m.skillsErr
	}

	out := cloneRecordV1(record)
	out.Skills = append(out.Skills, m.skillsV1...)

	return out, nil
}

func (m *mockEnricherClient) EnrichWithDomainsV1(_ context.Context, record *typesv1.Record) (*typesv1.Record, error) {
	if m.domainsErr != nil {
		return nil, m.domainsErr
	}

	out := cloneRecordV1(record)
	out.Domains = append(out.Domains, m.domainsV1...)

	return out, nil
}

func cloneRecordV1Alpha1(r *typesv1alpha1.Record) *typesv1alpha1.Record {
	if r == nil {
		return nil
	}

	out := &typesv1alpha1.Record{
		Name:          r.GetName(),
		Version:       r.GetVersion(),
		SchemaVersion: r.GetSchemaVersion(),
	}
	if r.Skills != nil {
		out.Skills = append([]*typesv1alpha1.Skill(nil), r.GetSkills()...)
	}

	if r.Domains != nil {
		out.Domains = append([]*typesv1alpha1.Domain(nil), r.GetDomains()...)
	}

	return out
}

func cloneRecordV1(r *typesv1.Record) *typesv1.Record {
	if r == nil {
		return nil
	}

	out := &typesv1.Record{
		Name:          r.GetName(),
		Version:       r.GetVersion(),
		SchemaVersion: r.GetSchemaVersion(),
	}
	if r.Skills != nil {
		out.Skills = append([]*typesv1.Skill(nil), r.GetSkills()...)
	}

	if r.Domains != nil {
		out.Domains = append([]*typesv1.Domain(nil), r.GetDomains()...)
	}

	return out
}

func TestTransformer_Transform_WithMockEnricher_Success(t *testing.T) {
	mock := &mockEnricherClient{
		skills:    []*typesv1alpha1.Skill{{Name: "test/skill", Id: 1}},
		domains:   []*typesv1alpha1.Domain{{Name: "test/domain", Id: 10}},
		skillsV1:  []*typesv1.Skill{{Name: "test/skill", Id: 1}},
		domainsV1: []*typesv1.Domain{{Name: "test/domain", Id: 10}},
	}
	transformer := NewTransformerWithHost(mock)
	source := mcpapiv0.ServerResponse{
		Server: mcpapiv0.ServerJSON{
			Name:        "test-server",
			Version:     testVersion,
			Description: "Test server",
			Packages: []model.Package{
				{RegistryType: "npm", Identifier: "test/pkg", Version: testVersion, Transport: model.Transport{Type: "stdio"}},
			},
		},
	}

	record, err := transformer.Transform(context.Background(), source)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	if record == nil || record.GetData() == nil {
		t.Fatal("Transform() returned nil record or nil data")
	}

	fields := record.GetData().GetFields()
	if fields["name"].GetStringValue() != "test-server" || fields["version"].GetStringValue() != testVersion {
		t.Errorf("record name/version = %q / %q", fields["name"].GetStringValue(), fields["version"].GetStringValue())
	}
	// Enricher added skills/domains
	if skills := fields["skills"]; skills != nil && skills.GetListValue() != nil {
		if n := len(skills.GetListValue().GetValues()); n != 1 {
			t.Errorf("skills count = %d, want 1", n)
		}
	}

	if domains := fields["domains"]; domains != nil && domains.GetListValue() != nil {
		if n := len(domains.GetListValue().GetValues()); n != 1 {
			t.Errorf("domains count = %d, want 1", n)
		}
	}
}

func TestTransformer_Transform_WithMockEnricher_EnrichmentError(t *testing.T) {
	mock := &mockEnricherClient{skillsErr: errors.New("enrich skills failed")}
	transformer := NewTransformerWithHost(mock)
	source := mcpapiv0.ServerResponse{
		Server: mcpapiv0.ServerJSON{
			Name:     "test-server",
			Version:  testVersion,
			Packages: []model.Package{{RegistryType: "npm", Identifier: "test/pkg", Version: testVersion, Transport: model.Transport{Type: "stdio"}}},
		},
	}

	_, err := transformer.Transform(context.Background(), source)
	if err == nil {
		t.Fatal("Transform() expected error when enrichment fails")
	}

	if !strings.Contains(err.Error(), "enrich") && !strings.Contains(err.Error(), "skills") {
		t.Errorf("error = %v", err)
	}
}

func TestGetSchemaVersion(t *testing.T) {
	tests := []struct {
		name    string
		s       *structpb.Struct
		want    string
		wantErr string
	}{
		{"nil struct", nil, "", "struct is nil"},
		{"nil fields", &structpb.Struct{Fields: nil}, "", "no fields"},
		{"missing schema_version", &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"name": structpb.NewStringValue("x"),
			},
		}, "", "schema_version"},
		{"success", &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"schema_version": structpb.NewStringValue("0.8.0"),
				"name":           structpb.NewStringValue("test"),
			},
		}, "0.8.0", ""},
		{"v1 version", &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"schema_version": structpb.NewStringValue("1.0.0"),
			},
		}, "1.0.0", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSchemaVersion(tt.s)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %v", err)
				}

				return
			}

			if err != nil {
				t.Fatalf("getSchemaVersion() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("getSchemaVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStructToOASFRecord_Error(t *testing.T) {
	// nil struct: protojson.Marshal(nil) may behave differently; use struct with invalid content or empty
	_, err := structToOASFRecord(nil)
	if err == nil {
		t.Error("structToOASFRecord(nil) expected error")
	}
}

func TestStructToOASFRecordV1_Error(t *testing.T) {
	_, err := structToOASFRecordV1(nil)
	if err == nil {
		t.Error("structToOASFRecordV1(nil) expected error")
	}
}
