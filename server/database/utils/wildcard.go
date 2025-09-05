// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
)

// ContainsWildcards checks if a pattern contains wildcard characters (* or ? or []).
func ContainsWildcards(pattern string) bool {
	return strings.Contains(pattern, "*") || strings.Contains(pattern, "?") || containsListWildcard(pattern)
}

// containsListWildcard checks if a pattern contains list wildcard characters [].
func containsListWildcard(pattern string) bool {
	openIdx := strings.Index(pattern, "[")
	closeIdx := strings.Index(pattern, "]")

	return openIdx != -1 && closeIdx > openIdx
}

// BuildWildcardCondition builds a WHERE condition for wildcard or exact matching.
// Returns the condition string and arguments for the WHERE clause.
func BuildWildcardCondition(field string, patterns []string) (string, []interface{}) {
	if len(patterns) == 0 {
		return "", nil
	}

	conditions := make([]string, 0, len(patterns))
	args := make([]interface{}, 0, len(patterns))

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
func BuildSingleWildcardCondition(field, pattern string) (string, string) {
	if ContainsWildcards(pattern) {
		return "LOWER(" + field + ") GLOB ?", strings.ToLower(pattern)
	}

	return "LOWER(" + field + ") = ?", strings.ToLower(pattern)
}
