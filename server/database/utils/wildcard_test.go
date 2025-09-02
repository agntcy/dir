// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"reflect"
	"testing"
)

func TestContainsWildcards(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected bool
	}{
		{
			name:     "no wildcards",
			pattern:  "simple",
			expected: false,
		},
		{
			name:     "single asterisk",
			pattern:  "test*",
			expected: true,
		},
		{
			name:     "asterisk at beginning",
			pattern:  "*test",
			expected: true,
		},
		{
			name:     "asterisk in middle",
			pattern:  "te*st",
			expected: true,
		},
		{
			name:     "multiple asterisks",
			pattern:  "*test*",
			expected: true,
		},
		{
			name:     "question mark (not a wildcard)",
			pattern:  "test?",
			expected: false,
		},
		{
			name:     "mixed asterisk and question mark",
			pattern:  "test*?",
			expected: true,
		},
		{
			name:     "empty string",
			pattern:  "",
			expected: false,
		},
		{
			name:     "only asterisk",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "complex pattern",
			pattern:  "api-*-v2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsWildcards(tt.pattern)
			if result != tt.expected {
				t.Errorf("ContainsWildcards(%q) = %v, want %v", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestWildcardToSQL(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "no wildcards",
			pattern:  "simple",
			expected: "simple",
		},
		{
			name:     "single asterisk",
			pattern:  "test*",
			expected: "test%",
		},
		{
			name:     "asterisk at beginning",
			pattern:  "*test",
			expected: "%test",
		},
		{
			name:     "asterisk in middle",
			pattern:  "te*st",
			expected: "te%st",
		},
		{
			name:     "multiple asterisks",
			pattern:  "*test*",
			expected: "%test%",
		},
		{
			name:     "question mark (literal)",
			pattern:  "test?",
			expected: "test?",
		},
		{
			name:     "mixed asterisk and question mark",
			pattern:  "test*?",
			expected: "test%?",
		},
		{
			name:     "only asterisk",
			pattern:  "*",
			expected: "%",
		},
		{
			name:     "complex pattern",
			pattern:  "api-*-v2",
			expected: "api-%-v2",
		},
		{
			name:     "escape existing percent",
			pattern:  "test%*",
			expected: "test\\%%",
		},
		{
			name:     "escape existing underscore",
			pattern:  "test_*",
			expected: "test\\_%",
		},
		{
			name:     "escape both percent and underscore",
			pattern:  "test%_*",
			expected: "test\\%\\_%",
		},
		{
			name:     "no wildcards with SQL chars",
			pattern:  "test%_",
			expected: "test%_",
		},
		{
			name:     "empty string",
			pattern:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WildcardToSQL(tt.pattern)
			if result != tt.expected {
				t.Errorf("WildcardToSQL(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestBuildWildcardCondition(t *testing.T) {
	tests := []struct {
		name              string
		field             string
		patterns          []string
		expectedCondition string
		expectedArgs      []interface{}
	}{
		{
			name:              "empty patterns",
			field:             "field",
			patterns:          []string{},
			expectedCondition: "",
			expectedArgs:      nil,
		},
		{
			name:              "single exact pattern",
			field:             "name",
			patterns:          []string{"test"},
			expectedCondition: "name = ?",
			expectedArgs:      []interface{}{"test"},
		},
		{
			name:              "single wildcard pattern",
			field:             "name",
			patterns:          []string{"test*"},
			expectedCondition: "name LIKE ?",
			expectedArgs:      []interface{}{"test%"},
		},
		{
			name:              "multiple exact patterns",
			field:             "name",
			patterns:          []string{"test1", "test2"},
			expectedCondition: "(name = ? OR name = ?)",
			expectedArgs:      []interface{}{"test1", "test2"},
		},
		{
			name:              "multiple wildcard patterns",
			field:             "name",
			patterns:          []string{"test*", "*service"},
			expectedCondition: "(name LIKE ? OR name LIKE ?)",
			expectedArgs:      []interface{}{"test%", "%service"},
		},
		{
			name:              "mixed exact and wildcard patterns",
			field:             "name",
			patterns:          []string{"python*", "go", "java*"},
			expectedCondition: "(name LIKE ? OR name = ? OR name LIKE ?)",
			expectedArgs:      []interface{}{"python%", "go", "java%"},
		},
		{
			name:              "single pattern no parentheses",
			field:             "version",
			patterns:          []string{"v1.*"},
			expectedCondition: "version LIKE ?",
			expectedArgs:      []interface{}{"v1.%"},
		},
		{
			name:              "complex field name",
			field:             "skills.name",
			patterns:          []string{"*script"},
			expectedCondition: "skills.name LIKE ?",
			expectedArgs:      []interface{}{"%script"},
		},
		{
			name:              "pattern with SQL injection chars",
			field:             "name",
			patterns:          []string{"test%_*"},
			expectedCondition: "name LIKE ?",
			expectedArgs:      []interface{}{"test\\%\\_%"},
		},
		{
			name:              "question mark as literal",
			field:             "name",
			patterns:          []string{"test?", "pattern*"},
			expectedCondition: "(name = ? OR name LIKE ?)",
			expectedArgs:      []interface{}{"test?", "pattern%"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, args := BuildWildcardCondition(tt.field, tt.patterns)

			if condition != tt.expectedCondition {
				t.Errorf("BuildWildcardCondition(%q, %v) condition = %q, want %q",
					tt.field, tt.patterns, condition, tt.expectedCondition)
			}

			if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("BuildWildcardCondition(%q, %v) args = %v, want %v",
					tt.field, tt.patterns, args, tt.expectedArgs)
			}
		})
	}
}

func TestWildcardIntegration(t *testing.T) {
	// Test the integration of all functions together
	tests := []struct {
		name              string
		field             string
		patterns          []string
		expectedCondition string
		expectedArgs      []interface{}
	}{
		{
			name:              "real world example - skill names",
			field:             "skills.name",
			patterns:          []string{"python*", "javascript", "*script", "go"},
			expectedCondition: "(skills.name LIKE ? OR skills.name = ? OR skills.name LIKE ? OR skills.name = ?)",
			expectedArgs:      []interface{}{"python%", "javascript", "%script", "go"},
		},
		{
			name:              "real world example - locator types",
			field:             "locators.type",
			patterns:          []string{"http*", "ftp*", "file"},
			expectedCondition: "(locators.type LIKE ? OR locators.type LIKE ? OR locators.type = ?)",
			expectedArgs:      []interface{}{"http%", "ftp%", "file"},
		},
		{
			name:              "real world example - extension names",
			field:             "extensions.name",
			patterns:          []string{"*-plugin", "*-extension", "core"},
			expectedCondition: "(extensions.name LIKE ? OR extensions.name LIKE ? OR extensions.name = ?)",
			expectedArgs:      []interface{}{"%-plugin", "%-extension", "core"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, args := BuildWildcardCondition(tt.field, tt.patterns)

			if condition != tt.expectedCondition {
				t.Errorf("Integration test %q: condition = %q, want %q",
					tt.name, condition, tt.expectedCondition)
			}

			if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("Integration test %q: args = %v, want %v",
					tt.name, args, tt.expectedArgs)
			}
		})
	}
}

// Benchmark tests to ensure performance is acceptable.
func BenchmarkContainsWildcards(b *testing.B) {
	patterns := []string{
		"simple",
		"test*",
		"*test",
		"te*st",
		"*test*",
		"complex-pattern-*-with-multiple-*-wildcards",
	}

	b.ResetTimer()

	for range b.N {
		for _, pattern := range patterns {
			ContainsWildcards(pattern)
		}
	}
}

func BenchmarkWildcardToSQL(b *testing.B) {
	patterns := []string{
		"simple",
		"test*",
		"*test",
		"te*st",
		"*test*",
		"test%_*",
		"complex-pattern-*-with-multiple-*-wildcards",
	}

	b.ResetTimer()

	for range b.N {
		for _, pattern := range patterns {
			WildcardToSQL(pattern)
		}
	}
}

func BenchmarkBuildWildcardCondition(b *testing.B) {
	patterns := []string{"python*", "go", "java*", "*script", "typescript"}
	field := "skills.name"

	b.ResetTimer()

	for range b.N {
		BuildWildcardCondition(field, patterns)
	}
}
