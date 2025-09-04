// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"strings"
)

// ContainsWildcards checks if a pattern contains wildcard characters (*).
func ContainsWildcards(pattern string) bool {
	return strings.Contains(pattern, "*")
}

// BuildWildcardCondition builds a WHERE condition for wildcard or exact matching.
// Returns the condition string and arguments for the WHERE clause.
func BuildWildcardCondition(field string, patterns []string) (string, []interface{}) {
	if len(patterns) == 0 {
		return "", nil
	}

	var conditions []string

	var args []interface{}

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
func BuildSingleWildcardCondition(field, pattern string) (string, interface{}) {
	if ContainsWildcards(pattern) {
		return "LOWER(" + field + ") GLOB ?", strings.ToLower(pattern)
	}
	return "LOWER(" + field + ") = ?", strings.ToLower(pattern)
}
