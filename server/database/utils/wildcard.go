// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
	"strings"
)

// Constants for GLOB pattern escaping
const (
	// GLOB special characters that need escaping
	globLiteralAsterisk     = "[*]"
	globLiteralQuestion     = "[?]"
	globLiteralBracketOpen  = "[[]"
	globLiteralBracketClose = "[]]"

	// Growth factor for strings.Builder pre-allocation
	buildGrowthFactor = 2

	// SQL operators
	sqlGlobLower  = "LOWER(%s) GLOB LOWER(?)"
	sqlEqualLower = "LOWER(%s) = LOWER(?)"
)

// ContainsWildcards checks if a pattern contains unescaped wildcard characters (*).
// Escaped asterisks (\*) are not considered wildcards.
func ContainsWildcards(s string) bool {
	for i := 0; i < len(s); {
		ch := s[i]

		// Handle escapes from user input
		if ch == '\\' {
			if i+1 < len(s) {
				// Skip the escaped character (including \*)
				i += 2
				continue
			}
			// Trailing backslash
			i++
			continue
		}

		// Check for unescaped wildcard
		if ch == '*' {
			return true
		}

		i++
	}
	return false
}

// UserGlobOnlyStar converts a user pattern where only '*' is a wildcard.
// Rules:
// - Unescaped '*'  => '*' (wildcard many)
// - Escaped '\*'   => literal '*'  (-> "[*]")
// - '?'            => literal '?'  (-> "[?]")
// - '['            => literal '['  (-> "[[]")
// - ']'            => literal ']'  (-> "[]]")
// - '\?' '\[' '\]' => their literal forms as above
// - '\' not before a recognized char => kept as literal '\'
//
// Performance: O(n) time complexity, pre-allocates buffer for efficiency.
// Use with: "... GLOB ?" for case-sensitive or "LOWER(field) GLOB LOWER(?)" for case-insensitive.
func UserGlobOnlyStar(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	b.Grow(len(s) * buildGrowthFactor)

	for i := 0; i < len(s); {
		ch := s[i]

		if ch == '\\' {
			if i+1 < len(s) {
				n := s[i+1]
				switch n {
				case '*':
					b.WriteString(globLiteralAsterisk)
				case '?':
					b.WriteString(globLiteralQuestion)
				case '[':
					b.WriteString(globLiteralBracketOpen)
				case ']':
					b.WriteString(globLiteralBracketClose)
				case '\\':
					b.WriteString(`\\`)
				default:
					// Unknown escape: keep backslash and char literally
					b.WriteByte('\\')
					b.WriteByte(n)
				}
				i += 2
				continue
			}
			// trailing backslash -> literal backslash
			b.WriteByte('\\')
			i++
			continue
		}

		switch ch {
		case '*':
			b.WriteByte('*')
		case '?':
			b.WriteString(globLiteralQuestion)
		case '[':
			b.WriteString(globLiteralBracketOpen)
		case ']':
			b.WriteString(globLiteralBracketClose)
		default:
			b.WriteByte(ch)
		}
		i++
	}
	return b.String()
}

// BuildWildcardCondition builds a WHERE condition for wildcard or exact matching using GLOB.
// Returns the condition string and arguments for the WHERE clause.
// Uses case-insensitive matching with LOWER().
//
// Performance: Pre-allocates slices with known capacity for efficiency.
// Field parameter should be properly sanitized before calling this function.
func BuildWildcardCondition(field string, patterns []string) (string, []interface{}) {
	if len(patterns) == 0 || field == "" {
		return "", nil
	}

	// Pre-allocate slices with known capacity to avoid reallocations
	conditions := make([]string, 0, len(patterns))
	args := make([]interface{}, 0, len(patterns))

	for _, pattern := range patterns {
		// Skip empty patterns
		if pattern == "" {
			continue
		}

		if ContainsWildcards(pattern) {
			conditions = append(conditions, fmt.Sprintf(sqlGlobLower, field))
			args = append(args, UserGlobOnlyStar(pattern))
		} else {
			conditions = append(conditions, fmt.Sprintf(sqlEqualLower, field))
			args = append(args, pattern)
		}
	}

	// Handle case where all patterns were empty
	if len(conditions) == 0 {
		return "", nil
	}

	condition := strings.Join(conditions, " OR ")
	if len(conditions) > 1 {
		condition = "(" + condition + ")"
	}

	return condition, args
}
