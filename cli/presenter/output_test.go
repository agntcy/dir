// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package presenter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
)

// Test constants for frequently used values with semantic meaning.
const (
	outputFlagName       = "output"
	outputShortFlagName  = "o"
	emptyJSONArray       = "[]\n"
	defaultFormatValue   = "human"
	flagNotFoundMsg      = "--output flag not found"
	shortFlagNotFoundMsg = "short flag -o not found"
)

func TestGetOutputOptions(t *testing.T) {
	tests := []struct {
		name           string
		flagValue      string
		expectedFormat OutputFormat
	}{
		{
			name:           "default format",
			flagValue:      "",
			expectedFormat: FormatHuman,
		},
		{
			name:           "human format",
			flagValue:      string(FormatHuman),
			expectedFormat: FormatHuman,
		},
		{
			name:           "json format",
			flagValue:      string(FormatJSON),
			expectedFormat: FormatJSON,
		},
		{
			name:           "jsonl format",
			flagValue:      string(FormatJSONL),
			expectedFormat: FormatJSONL,
		},
		{
			name:           "raw format",
			flagValue:      string(FormatRaw),
			expectedFormat: FormatRaw,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			AddOutputFlags(cmd)

			if tt.flagValue != "" {
				if err := cmd.Flags().Set(outputFlagName, tt.flagValue); err != nil {
					t.Fatalf("failed to set flag: %v", err)
				}
			}

			opts := GetOutputOptions(cmd)
			if opts.Format != tt.expectedFormat {
				t.Errorf("expected format %q, got %q", tt.expectedFormat, opts.Format)
			}
		})
	}
}

func TestIsStructuredOutput(t *testing.T) {
	tests := []struct {
		name       string
		format     OutputFormat
		isStructed bool
	}{
		{
			name:       "human is not structured",
			format:     FormatHuman,
			isStructed: false,
		},
		{
			name:       "json is structured",
			format:     FormatJSON,
			isStructed: true,
		},
		{
			name:       "jsonl is structured",
			format:     FormatJSONL,
			isStructed: true,
		},
		{
			name:       "raw is structured",
			format:     FormatRaw,
			isStructed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := OutputOptions{Format: tt.format}
			if got := opts.IsStructuredOutput(); got != tt.isStructed {
				t.Errorf("IsStructuredOutput() = %v, want %v", got, tt.isStructed)
			}
		})
	}
}

func TestPrintMessageHumanFormat(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		message  string
		value    any
		expected string
	}{
		{
			name:     "simple string value",
			title:    "result",
			message:  "Found result",
			value:    "test-value",
			expected: "Found result: test-value\n",
		},
		{
			name:     "nil value",
			title:    "results",
			message:  "Found results",
			value:    nil,
			expected: "No results found\n",
		},
		{
			name:     "empty slice",
			title:    "items",
			message:  "Found items",
			value:    []interface{}{},
			expected: "No items found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetOut(&buf)
			AddOutputFlags(cmd)

			err := PrintMessage(cmd, tt.title, tt.message, tt.value)
			if err != nil {
				t.Fatalf("PrintMessage() error = %v", err)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("PrintMessage() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrintMessageJSONFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:  "simple object",
			value: map[string]string{"key": "value"},
			expected: `{
  "key": "value"
}
`,
		},
		{
			name:  "array of objects",
			value: []map[string]string{{"id": "1"}, {"id": "2"}},
			expected: `[
  {
    "id": "1"
  },
  {
    "id": "2"
  }
]
`,
		},
		{
			name:     "nil value",
			value:    nil,
			expected: emptyJSONArray,
		},
		{
			name:     "empty slice",
			value:    []interface{}{},
			expected: emptyJSONArray,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetOut(&buf)
			AddOutputFlags(cmd)

			if err := cmd.Flags().Set(outputFlagName, string(FormatJSON)); err != nil {
				t.Fatalf("failed to set flag: %v", err)
			}

			err := PrintMessage(cmd, "test", "Test", tt.value)
			if err != nil {
				t.Fatalf("PrintMessage() error = %v", err)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("PrintMessage() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrintMessageJSONLFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "single object",
			value:    map[string]string{"key": "value"},
			expected: "{\"key\":\"value\"}\n",
		},
		{
			name:     "array of objects",
			value:    []map[string]string{{"id": "1"}, {"id": "2"}},
			expected: "{\"id\":\"1\"}\n{\"id\":\"2\"}\n",
		},
		{
			name:     "array of strings",
			value:    []string{"a", "b", "c"},
			expected: "\"a\"\n\"b\"\n\"c\"\n",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: emptyJSONArray,
		},
		{
			name:     "empty slice",
			value:    []interface{}{},
			expected: emptyJSONArray,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetOut(&buf)
			AddOutputFlags(cmd)

			if err := cmd.Flags().Set(outputFlagName, string(FormatJSONL)); err != nil {
				t.Fatalf("failed to set flag: %v", err)
			}

			err := PrintMessage(cmd, "test", "Test", tt.value)
			if err != nil {
				t.Fatalf("PrintMessage() error = %v", err)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("PrintMessage() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrintMessageRawFormat(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "string value",
			value:    "test-cid-123",
			expected: "test-cid-123",
		},
		{
			name:     "slice of strings",
			value:    []string{"cid1", "cid2"},
			expected: "[cid1 cid2]",
		},
		{
			name:     "nil value",
			value:    nil,
			expected: emptyJSONArray,
		},
		{
			name:     "empty slice",
			value:    []interface{}{},
			expected: emptyJSONArray,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetOut(&buf)
			AddOutputFlags(cmd)

			if err := cmd.Flags().Set(outputFlagName, string(FormatRaw)); err != nil {
				t.Fatalf("failed to set flag: %v", err)
			}

			err := PrintMessage(cmd, "test", "Test", tt.value)
			if err != nil {
				t.Fatalf("PrintMessage() error = %v", err)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("PrintMessage() output = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPrintJSONL(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "single object",
			value:    map[string]string{"key": "value"},
			expected: "{\"key\":\"value\"}\n",
		},
		{
			name:     "array of objects",
			value:    []map[string]int{{"count": 1}, {"count": 2}},
			expected: "{\"count\":1}\n{\"count\":2}\n",
		},
		{
			name: "complex nested object",
			value: map[string]interface{}{
				"id":   "123",
				"tags": []string{"a", "b"},
			},
			expected: "{\"id\":\"123\",\"tags\":[\"a\",\"b\"]}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			cmd := &cobra.Command{}
			cmd.SetOut(&buf)

			err := printJSONL(cmd, tt.value)
			if err != nil {
				t.Fatalf("printJSONL() error = %v", err)
			}

			if got := buf.String(); got != tt.expected {
				t.Errorf("printJSONL() output = %q, want %q", got, tt.expected)
			}

			// Verify each line is valid JSON
			lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
			for i, line := range lines {
				var v interface{}
				if err := json.Unmarshal(line, &v); err != nil {
					t.Errorf("line %d is not valid JSON: %v", i, err)
				}
			}
		})
	}
}

func TestIsSliceOrArray(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: false,
		},
		{
			name:     "string value",
			value:    "test",
			expected: false,
		},
		{
			name:     "int value",
			value:    42,
			expected: false,
		},
		{
			name:     "map value",
			value:    map[string]string{"key": "value"},
			expected: false,
		},
		{
			name:     "slice of strings",
			value:    []string{"a", "b"},
			expected: true,
		},
		{
			name:     "slice of interfaces",
			value:    []interface{}{"a", 1},
			expected: true,
		},
		{
			name:     "array of ints",
			value:    [3]int{1, 2, 3},
			expected: true,
		},
		{
			name:     "empty slice",
			value:    []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSliceOrArray(tt.value); got != tt.expected {
				t.Errorf("isSliceOrArray() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsEmptySlice(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: false,
		},
		{
			name:     "non-slice value",
			value:    "test",
			expected: false,
		},
		{
			name:     "empty interface slice",
			value:    []interface{}{},
			expected: true,
		},
		{
			name:     "non-empty interface slice",
			value:    []interface{}{"item"},
			expected: false,
		},
		{
			name:     "empty string slice",
			value:    []string{},
			expected: true,
		},
		{
			name:     "non-empty string slice",
			value:    []string{"item"},
			expected: false,
		},
		{
			name:     "empty array",
			value:    [0]int{},
			expected: true,
		},
		{
			name:     "non-empty array",
			value:    [1]int{42},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isEmptySlice(tt.value); got != tt.expected {
				t.Errorf("isEmptySlice() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAddOutputFlags(t *testing.T) {
	cmd := &cobra.Command{}
	AddOutputFlags(cmd)

	// Check flag exists
	flag := cmd.Flags().Lookup(outputFlagName)
	if flag == nil {
		t.Fatal(flagNotFoundMsg)
	}

	// Check short flag exists
	if shortFlag := cmd.Flags().ShorthandLookup(outputShortFlagName); shortFlag == nil {
		t.Error(shortFlagNotFoundMsg)
	}

	// Check default value
	if flag.DefValue != defaultFormatValue {
		t.Errorf("default value = %q, want %q", flag.DefValue, defaultFormatValue)
	}

	// Check usage message contains all formats
	usage := flag.Usage

	formats := []string{string(FormatHuman), string(FormatJSON), string(FormatJSONL), string(FormatRaw)}
	for _, format := range formats {
		if !containsSubstring(usage, format) {
			t.Errorf("usage message missing format %q", format)
		}
	}
}

func TestPrintMessageMarshalError(t *testing.T) {
	// Create a value that can't be marshaled to JSON
	invalidValue := make(chan int)

	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	AddOutputFlags(cmd)

	if err := cmd.Flags().Set(outputFlagName, string(FormatJSON)); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := PrintMessage(cmd, "test", "Test", invalidValue)
	if err == nil {
		t.Error("PrintMessage() expected error for invalid JSON, got nil")
	}
}

func TestPrintMessageJSONLMarshalError_SingleObject(t *testing.T) {
	// Create a value that can't be marshaled to JSON (channel)
	invalidValue := make(chan int)

	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	AddOutputFlags(cmd)

	if err := cmd.Flags().Set(outputFlagName, string(FormatJSONL)); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := PrintMessage(cmd, "test", "Test", invalidValue)
	if err == nil {
		t.Error("PrintMessage() expected error for invalid JSONL single object, got nil")
	}

	if err != nil && !containsSubstring(err.Error(), "failed to marshal JSON") {
		t.Errorf("PrintMessage() error = %v, should contain 'failed to marshal JSON'", err)
	}
}

func TestPrintMessageJSONLMarshalError_ArrayElement(t *testing.T) {
	// Create an array with an element that can't be marshaled to JSON
	invalidValue := []interface{}{
		map[string]interface{}{"id": 1, "name": "valid"},
		make(chan int), // This will cause marshal error
	}

	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	AddOutputFlags(cmd)

	if err := cmd.Flags().Set(outputFlagName, string(FormatJSONL)); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	err := PrintMessage(cmd, "test", "Test", invalidValue)
	if err == nil {
		t.Error("PrintMessage() expected error for invalid JSONL array element, got nil")
	}

	if err != nil && !containsSubstring(err.Error(), "failed to marshal JSON") {
		t.Errorf("PrintMessage() error = %v, should contain 'failed to marshal JSON'", err)
	}
}

func TestPrintMessageInvalidFormat(t *testing.T) {
	// Test with an invalid format (not one of the defined constants)
	var buf bytes.Buffer

	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	AddOutputFlags(cmd)

	// Set an invalid format value directly
	if err := cmd.Flags().Set(outputFlagName, "invalid-format"); err != nil {
		t.Fatalf("failed to set flag: %v", err)
	}

	// Should handle gracefully (return nil without error)
	err := PrintMessage(cmd, "test", "Test", "value")
	if err != nil {
		t.Errorf("PrintMessage() with invalid format should not error, got: %v", err)
	}
}

// Helper function.
func containsSubstring(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
