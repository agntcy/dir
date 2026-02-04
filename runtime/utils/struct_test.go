// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wsl_v5
package utils

import (
	"testing"
)

func TestInterfaceToStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr bool
	}{
		{
			name: "converts simple map",
			input: map[string]any{
				"key": "value",
				"num": 42,
			},
			wantErr: false,
		},
		{
			name: "converts nested map",
			input: map[string]any{
				"outer": map[string]any{
					"inner": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "converts struct",
			input: struct {
				Name  string `json:"name"`
				Count int    `json:"count"`
			}{
				Name:  "test",
				Count: 10,
			},
			wantErr: false,
		},
		{
			name:    "converts empty map",
			input:   map[string]any{},
			wantErr: false,
		},
		{
			name:    "converts nil",
			input:   nil,
			wantErr: false,
		},
		{
			name: "converts map with slice",
			input: map[string]any{
				"items": []string{"a", "b", "c"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterfaceToStruct(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("InterfaceToStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && tt.input != nil && result == nil {
				t.Error("InterfaceToStruct() returned nil for non-nil input")
			}
		})
	}
}

func TestInterfaceToStructPreservesValues(t *testing.T) {
	input := map[string]any{
		"string_val": "hello",
		"int_val":    float64(42), // JSON numbers are float64
		"bool_val":   true,
		"nested": map[string]any{
			"inner": "world",
		},
	}

	result, err := InterfaceToStruct(input)
	if err != nil {
		t.Fatalf("InterfaceToStruct() error = %v", err)
	}

	fields := result.GetFields()

	if fields["string_val"].GetStringValue() != "hello" {
		t.Errorf("string_val = %v, want 'hello'", fields["string_val"].GetStringValue())
	}

	if fields["int_val"].GetNumberValue() != 42 {
		t.Errorf("int_val = %v, want 42", fields["int_val"].GetNumberValue())
	}

	if fields["bool_val"].GetBoolValue() != true {
		t.Errorf("bool_val = %v, want true", fields["bool_val"].GetBoolValue())
	}

	nested := fields["nested"].GetStructValue()
	if nested == nil {
		t.Fatal("nested struct is nil")
	}
	if nested.GetFields()["inner"].GetStringValue() != "world" {
		t.Errorf("nested.inner = %v, want 'world'", nested.GetFields()["inner"].GetStringValue())
	}
}
