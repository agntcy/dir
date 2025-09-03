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
			expectedCondition: "LOWER(name) = LOWER(?)",
			expectedArgs:      []interface{}{"test"},
		},
		{
			name:              "single wildcard pattern",
			field:             "name",
			patterns:          []string{"test*"},
			expectedCondition: "LOWER(name) GLOB LOWER(?)",
			expectedArgs:      []interface{}{"test*"},
		},
		{
			name:              "multiple exact patterns",
			field:             "name",
			patterns:          []string{"test1", "test2"},
			expectedCondition: "(LOWER(name) = LOWER(?) OR LOWER(name) = LOWER(?))",
			expectedArgs:      []interface{}{"test1", "test2"},
		},
		{
			name:              "multiple wildcard patterns",
			field:             "name",
			patterns:          []string{"test*", "*service"},
			expectedCondition: "(LOWER(name) GLOB LOWER(?) OR LOWER(name) GLOB LOWER(?))",
			expectedArgs:      []interface{}{"test*", "*service"},
		},
		{
			name:              "mixed exact and wildcard patterns",
			field:             "name",
			patterns:          []string{"python*", "go", "java*"},
			expectedCondition: "(LOWER(name) GLOB LOWER(?) OR LOWER(name) = LOWER(?) OR LOWER(name) GLOB LOWER(?))",
			expectedArgs:      []interface{}{"python*", "go", "java*"},
		},
		{
			name:              "single pattern no parentheses",
			field:             "version",
			patterns:          []string{"v1.*"},
			expectedCondition: "LOWER(version) GLOB LOWER(?)",
			expectedArgs:      []interface{}{"v1.*"},
		},
		{
			name:              "complex field name",
			field:             "skills.name",
			patterns:          []string{"*script"},
			expectedCondition: "LOWER(skills.name) GLOB LOWER(?)",
			expectedArgs:      []interface{}{"*script"},
		},
		{
			name:              "pattern with GLOB chars",
			field:             "name",
			patterns:          []string{"test?[a-z]*"},
			expectedCondition: "LOWER(name) GLOB LOWER(?)",
			expectedArgs:      []interface{}{"test[?][[]a-z[]]*"},
		},
		{
			name:              "escaped asterisk as literal",
			field:             "name",
			patterns:          []string{"test\\*", "pattern*"},
			expectedCondition: "(LOWER(name) = LOWER(?) OR LOWER(name) GLOB LOWER(?))",
			expectedArgs:      []interface{}{"test\\*", "pattern*"},
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

func TestGlobIntegration(t *testing.T) {
	// Test the integration of all functions together with GLOB
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
			expectedCondition: "(LOWER(skills.name) GLOB LOWER(?) OR LOWER(skills.name) = LOWER(?) OR LOWER(skills.name) GLOB LOWER(?) OR LOWER(skills.name) = LOWER(?))",
			expectedArgs:      []interface{}{"python*", "javascript", "*script", "go"},
		},
		{
			name:              "real world example - locator types",
			field:             "locators.type",
			patterns:          []string{"http*", "ftp*", "file"},
			expectedCondition: "(LOWER(locators.type) GLOB LOWER(?) OR LOWER(locators.type) GLOB LOWER(?) OR LOWER(locators.type) = LOWER(?))",
			expectedArgs:      []interface{}{"http*", "ftp*", "file"},
		},
		{
			name:              "real world example - extension names",
			field:             "extensions.name",
			patterns:          []string{"*-plugin", "*-extension", "core"},
			expectedCondition: "(LOWER(extensions.name) GLOB LOWER(?) OR LOWER(extensions.name) GLOB LOWER(?) OR LOWER(extensions.name) = LOWER(?))",
			expectedArgs:      []interface{}{"*-plugin", "*-extension", "core"},
		},
		{
			name:              "version patterns",
			field:             "version",
			patterns:          []string{"v1.*", "v2.0", "*-beta"},
			expectedCondition: "(LOWER(version) GLOB LOWER(?) OR LOWER(version) = LOWER(?) OR LOWER(version) GLOB LOWER(?))",
			expectedArgs:      []interface{}{"v1.*", "v2.0", "*-beta"},
		},
		{
			name:              "escaped patterns",
			field:             "name",
			patterns:          []string{"\\*\\*bold\\*\\*", "normal*"},
			expectedCondition: "(LOWER(name) = LOWER(?) OR LOWER(name) GLOB LOWER(?))",
			expectedArgs:      []interface{}{"\\*\\*bold\\*\\*", "normal*"},
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

func BenchmarkUserGlobOnlyStar(b *testing.B) {
	patterns := []string{
		"simple",
		"test*",
		"*test",
		"te*st",
		"*test*",
		"test?[a-z]*",
		"complex-pattern-*-with-multiple-*-wildcards",
		"\\*\\*bold\\*\\*",
	}

	b.ResetTimer()

	for range b.N {
		for _, pattern := range patterns {
			UserGlobOnlyStar(pattern)
		}
	}
}

func TestUserGlobOnlyStar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		meaning  string
	}{
		{
			name:     "simple value",
			input:    "simple value",
			expected: "simple value",
			meaning:  "literal match (no wildcards)",
		},
		{
			name:     "asterisk wildcard",
			input:    "SIM*ple value",
			expected: "SIM*ple value",
			meaning:  "* remains as wildcard",
		},
		{
			name:     "GLOB special chars as literals",
			input:    "SIM?PLE [VALUE]",
			expected: "SIM[?]PLE [[]VALUE[]]",
			meaning:  "? [ ] treated as literals",
		},
		{
			name:     "multiple GLOB chars",
			input:    "ABC????[cd]????",
			expected: "ABC[?][?][?][?][[]cd[]][?][?][?][?]",
			meaning:  "all GLOB chars escaped",
		},
		{
			name:     "escaped asterisk",
			input:    "file\\*name",
			expected: "file[*]name",
			meaning:  "\\* becomes literal *",
		},
		{
			name:     "literal backslash",
			input:    "path\\\\dir",
			expected: "path\\\\dir",
			meaning:  "literal backslash",
		},
		{
			name:     "complex escaping with wildcard",
			input:    "test\\*pattern*end",
			expected: "test[*]pattern*end",
			meaning:  "\\* becomes literal *, * remains wildcard",
		},
		{
			name:     "escaped asterisk only",
			input:    "no\\*wildcards\\*here",
			expected: "no[*]wildcards[*]here",
			meaning:  "all \\* become literal *",
		},
		{
			name:     "escaped GLOB chars",
			input:    "test\\?pattern\\[data\\]",
			expected: "test[?]pattern[[]data[]]",
			meaning:  "escaped GLOB chars become literals",
		},
		{
			name:     "version pattern",
			input:    "version.v1.*",
			expected: "version.v1.*",
			meaning:  "version wildcard pattern",
		},
		{
			name:     "markdown bold",
			input:    "\\*\\*Bold Text\\*\\*",
			expected: "[*][*]Bold Text[*][*]",
			meaning:  "markdown asterisks as literals",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UserGlobOnlyStar(tt.input)
			if result != tt.expected {
				t.Errorf("UserGlobOnlyStar(%q) = %q, want %q (meaning: %s)",
					tt.input, result, tt.expected, tt.meaning)
			}
		})
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
