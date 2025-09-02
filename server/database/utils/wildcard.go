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

// WildcardToSQL converts wildcard patterns (*) to SQL LIKE patterns (%).
// It also escapes existing SQL wildcards to prevent injection.
func WildcardToSQL(pattern string) string {
	if !ContainsWildcards(pattern) {
		return pattern
	}

	// Escape existing SQL wildcards
	result := strings.ReplaceAll(pattern, "%", `\%`)
	result = strings.ReplaceAll(result, "_", `\_`)

	// Convert wildcard patterns to SQL patterns
	result = strings.ReplaceAll(result, "*", "%")

	return result
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
		if ContainsWildcards(pattern) {
			conditions = append(conditions, field+" LIKE ?")
			args = append(args, WildcardToSQL(pattern))
		} else {
			conditions = append(conditions, field+" = ?")
			args = append(args, pattern)
		}
	}

	condition := strings.Join(conditions, " OR ")
	if len(conditions) > 1 {
		condition = "(" + condition + ")"
	}

	return condition, args
}
