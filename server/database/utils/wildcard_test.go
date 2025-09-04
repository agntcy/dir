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
			name:     "question mark (wildcard in GLOB)",
			pattern:  "test?",
			expected: true,
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
		{
			name:     "only question mark",
			pattern:  "?",
			expected: true,
		},
		{
			name:     "multiple question marks",
			pattern:  "test???",
			expected: true,
		},
		{
			name:     "question mark at beginning",
			pattern:  "?test",
			expected: true,
		},
		{
			name:     "question mark in middle",
			pattern:  "te?st",
			expected: true,
		},
		{
			name:     "question mark at end",
			pattern:  "test?",
			expected: true,
		},
		{
			name:     "complex pattern with both wildcards",
			pattern:  "api-*-v?.?",
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

func TestBuildSingleWildcardCondition(t *testing.T) {
	tests := []struct {
		name              string
		field             string
		pattern           string
		expectedCondition string
		expectedArg       interface{}
	}{
		{
			name:              "exact match",
			field:             "name",
			pattern:           "Test",
			expectedCondition: "LOWER(name) = ?",
			expectedArg:       "test",
		},
		{
			name:              "wildcard with asterisk",
			field:             "name",
			pattern:           "Test*",
			expectedCondition: "LOWER(name) GLOB ?",
			expectedArg:       "test*",
		},
		{
			name:              "wildcard with question mark",
			field:             "version",
			pattern:           "V1.?",
			expectedCondition: "LOWER(version) GLOB ?",
			expectedArg:       "v1.?",
		},
		{
			name:              "complex field name",
			field:             "skills.name",
			pattern:           "*Script",
			expectedCondition: "LOWER(skills.name) GLOB ?",
			expectedArg:       "*script",
		},
		{
			name:              "wildcard with mixed asterisk and question mark",
			field:             "name",
			pattern:           "Test*?.txt",
			expectedCondition: "LOWER(name) GLOB ?",
			expectedArg:       "test*?.txt",
		},
		{
			name:              "multiple question marks",
			field:             "code",
			pattern:           "AB??-XY?",
			expectedCondition: "LOWER(code) GLOB ?",
			expectedArg:       "ab??-xy?",
		},
		{
			name:              "empty pattern",
			field:             "name",
			pattern:           "",
			expectedCondition: "LOWER(name) = ?",
			expectedArg:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, arg := BuildSingleWildcardCondition(tt.field, tt.pattern)

			if condition != tt.expectedCondition {
				t.Errorf("BuildSingleWildcardCondition(%q, %q) condition = %q, want %q",
					tt.field, tt.pattern, condition, tt.expectedCondition)
			}

			if arg != tt.expectedArg {
				t.Errorf("BuildSingleWildcardCondition(%q, %q) arg = %v, want %v",
					tt.field, tt.pattern, arg, tt.expectedArg)
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
			patterns:          []string{"Test"},
			expectedCondition: "LOWER(name) = ?",
			expectedArgs:      []interface{}{"test"},
		},
		{
			name:              "single wildcard pattern",
			field:             "name",
			patterns:          []string{"Test*"},
			expectedCondition: "LOWER(name) GLOB ?",
			expectedArgs:      []interface{}{"test*"},
		},
		{
			name:              "multiple exact patterns",
			field:             "name",
			patterns:          []string{"Test1", "Test2"},
			expectedCondition: "(LOWER(name) = ? OR LOWER(name) = ?)",
			expectedArgs:      []interface{}{"test1", "test2"},
		},
		{
			name:              "multiple wildcard patterns",
			field:             "name",
			patterns:          []string{"Test*", "*Service"},
			expectedCondition: "(LOWER(name) GLOB ? OR LOWER(name) GLOB ?)",
			expectedArgs:      []interface{}{"test*", "*service"},
		},
		{
			name:              "mixed exact and wildcard patterns",
			field:             "name",
			patterns:          []string{"Python*", "Go", "Java*"},
			expectedCondition: "(LOWER(name) GLOB ? OR LOWER(name) = ? OR LOWER(name) GLOB ?)",
			expectedArgs:      []interface{}{"python*", "go", "java*"},
		},
		{
			name:              "single pattern no parentheses",
			field:             "version",
			patterns:          []string{"V1.*"},
			expectedCondition: "LOWER(version) GLOB ?",
			expectedArgs:      []interface{}{"v1.*"},
		},
		{
			name:              "complex field name",
			field:             "skills.name",
			patterns:          []string{"*Script"},
			expectedCondition: "LOWER(skills.name) GLOB ?",
			expectedArgs:      []interface{}{"*script"},
		},
		{
			name:              "pattern with special chars (literal in GLOB)",
			field:             "name",
			patterns:          []string{"Test%_*"},
			expectedCondition: "LOWER(name) GLOB ?",
			expectedArgs:      []interface{}{"test%_*"},
		},
		{
			name:              "question mark as wildcard in GLOB",
			field:             "name",
			patterns:          []string{"Test?", "Pattern*"},
			expectedCondition: "(LOWER(name) GLOB ? OR LOWER(name) GLOB ?)",
			expectedArgs:      []interface{}{"test?", "pattern*"},
		},
		{
			name:              "multiple question marks in single pattern",
			field:             "version",
			patterns:          []string{"v?.?.?"},
			expectedCondition: "LOWER(version) GLOB ?",
			expectedArgs:      []interface{}{"v?.?.?"},
		},
		{
			name:              "mixed patterns with question marks",
			field:             "code",
			patterns:          []string{"AB??", "CD*", "EF", "GH?I"},
			expectedCondition: "(LOWER(code) GLOB ? OR LOWER(code) GLOB ? OR LOWER(code) = ? OR LOWER(code) GLOB ?)",
			expectedArgs:      []interface{}{"ab??", "cd*", "ef", "gh?i"},
		},
		{
			name:              "question mark with special characters",
			field:             "filename",
			patterns:          []string{"test?.txt", "data_?.csv"},
			expectedCondition: "(LOWER(filename) GLOB ? OR LOWER(filename) GLOB ?)",
			expectedArgs:      []interface{}{"test?.txt", "data_?.csv"},
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
			patterns:          []string{"Python*", "JavaScript", "*Script", "Go"},
			expectedCondition: "(LOWER(skills.name) GLOB ? OR LOWER(skills.name) = ? OR LOWER(skills.name) GLOB ? OR LOWER(skills.name) = ?)",
			expectedArgs:      []interface{}{"python*", "javascript", "*script", "go"},
		},
		{
			name:              "real world example - locator types",
			field:             "locators.type",
			patterns:          []string{"HTTP*", "FTP*", "File"},
			expectedCondition: "(LOWER(locators.type) GLOB ? OR LOWER(locators.type) GLOB ? OR LOWER(locators.type) = ?)",
			expectedArgs:      []interface{}{"http*", "ftp*", "file"},
		},
		{
			name:              "real world example - extension names",
			field:             "extensions.name",
			patterns:          []string{"*-Plugin", "*-Extension", "Core"},
			expectedCondition: "(LOWER(extensions.name) GLOB ? OR LOWER(extensions.name) GLOB ? OR LOWER(extensions.name) = ?)",
			expectedArgs:      []interface{}{"*-plugin", "*-extension", "core"},
		},
		{
			name:              "real world example - version patterns with question marks",
			field:             "version",
			patterns:          []string{"v?.0.0", "v1.?.?", "v2.*"},
			expectedCondition: "(LOWER(version) GLOB ? OR LOWER(version) GLOB ? OR LOWER(version) GLOB ?)",
			expectedArgs:      []interface{}{"v?.0.0", "v1.?.?", "v2.*"},
		},
		{
			name:              "real world example - file extensions with question marks",
			field:             "filename",
			patterns:          []string{"*.tx?", "data_?.csv", "log???.txt"},
			expectedCondition: "(LOWER(filename) GLOB ? OR LOWER(filename) GLOB ? OR LOWER(filename) GLOB ?)",
			expectedArgs:      []interface{}{"*.tx?", "data_?.csv", "log???.txt"},
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

func TestQuestionMarkWildcardFunctionality(t *testing.T) {
	tests := []struct {
		name              string
		field             string
		patterns          []string
		expectedCondition string
		expectedArgs      []interface{}
		description       string
	}{
		{
			name:              "single character replacement",
			field:             "code",
			patterns:          []string{"A?C"},
			expectedCondition: "LOWER(code) GLOB ?",
			expectedArgs:      []interface{}{"a?c"},
			description:       "? should match exactly one character",
		},
		{
			name:              "multiple single character replacements",
			field:             "serial",
			patterns:          []string{"AB??EF"},
			expectedCondition: "LOWER(serial) GLOB ?",
			expectedArgs:      []interface{}{"ab??ef"},
			description:       "Multiple ? should each match one character",
		},
		{
			name:              "question mark with asterisk combination",
			field:             "filename",
			patterns:          []string{"*.tx?", "data*.?sv"},
			expectedCondition: "(LOWER(filename) GLOB ? OR LOWER(filename) GLOB ?)",
			expectedArgs:      []interface{}{"*.tx?", "data*.?sv"},
			description:       "? and * should work together",
		},
		{
			name:              "question mark in version patterns",
			field:             "version",
			patterns:          []string{"v1.?.0", "v?.0.0"},
			expectedCondition: "(LOWER(version) GLOB ? OR LOWER(version) GLOB ?)",
			expectedArgs:      []interface{}{"v1.?.0", "v?.0.0"},
			description:       "? useful for version number wildcards",
		},
		{
			name:              "question mark with exact matches",
			field:             "type",
			patterns:          []string{"A?B", "exact", "C?D"},
			expectedCondition: "(LOWER(type) GLOB ? OR LOWER(type) = ? OR LOWER(type) GLOB ?)",
			expectedArgs:      []interface{}{"a?b", "exact", "c?d"},
			description:       "Mix of ? wildcards and exact matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition, args := BuildWildcardCondition(tt.field, tt.patterns)

			if condition != tt.expectedCondition {
				t.Errorf("%s: condition = %q, want %q", tt.description, condition, tt.expectedCondition)
			}

			if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("%s: args = %v, want %v", tt.description, args, tt.expectedArgs)
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
		"test?",
		"?test",
		"te?st",
		"test???",
		"*test?",
		"?test*",
		"complex-pattern-*-with-multiple-*-wildcards-and-?-marks",
	}

	b.ResetTimer()

	for range b.N {
		for _, pattern := range patterns {
			ContainsWildcards(pattern)
		}
	}
}

func BenchmarkBuildWildcardCondition(b *testing.B) {
	patterns := []string{"Python*", "Go", "Java*", "*Script", "TypeScript"}
	field := "skills.name"

	b.ResetTimer()

	for range b.N {
		BuildWildcardCondition(field, patterns)
	}
}
