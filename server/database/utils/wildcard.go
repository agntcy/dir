// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
)

// ContainsWildcards checks if a pattern contains wildcard characters (* or ?).
func ContainsWildcards(pattern string) bool {
	return strings.Contains(pattern, "*") || strings.Contains(pattern, "?")
}

// BuildWildcardCondition builds a WHERE condition for wildcard or exact matching.
// Returns the condition string and arguments for the WHERE clause.
func BuildWildcardCondition(field string, patterns []string) (string, []any) {
	if len(patterns) == 0 {
		return "", nil
	}

	conditions := make([]string, 0, len(patterns))
	args := make([]any, 0, len(patterns))

	for _, pattern := range patterns {
		condition, arg := BuildSingleWildcardCondition(field, pattern)
		conditions = append(conditions, condition)
		args = append(args, arg)
	}

	condition := strings.Join(conditions, " OR ")
	if len(conditions) > 1 {
		condition = "(" + condition + ")"
	}

	return condition, args
}

// BuildSingleWildcardCondition builds a WHERE condition for a single field with wildcard or exact matching.
// Returns the condition string and argument for the WHERE clause.
// Uses SQL LIKE which works with both SQLite and PostgreSQL.
//
// Supported wildcards:
//   - * matches any sequence of characters (converted to %)
//   - ? matches any single character (converted to _)
func BuildSingleWildcardCondition(field, pattern string) (string, string) {
	if !ContainsWildcards(pattern) {
		return "LOWER(" + field + ") = ?", strings.ToLower(pattern)
	}

	likePattern := convertGlobToLike(pattern)

	// Use explicit ESCAPE clause for cross-database compatibility
	// This ensures backslash escaping works in both SQLite and PostgreSQL
	return "LOWER(" + field + ") LIKE ? ESCAPE '\\'", strings.ToLower(likePattern)
}

// convertGlobToLike converts glob-style wildcards to SQL LIKE-style wildcards.
// * -> %
// ? -> _
// Also escapes existing % and _ characters in the pattern.
func convertGlobToLike(pattern string) string {
	// First escape existing LIKE special characters
	result := strings.ReplaceAll(pattern, "%", "\\%")
	result = strings.ReplaceAll(result, "_", "\\_")

	// Then convert glob wildcards to LIKE wildcards
	result = strings.ReplaceAll(result, "*", "%")
	result = strings.ReplaceAll(result, "?", "_")

	return result
}
