// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"fmt"
	"math"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
)

// toPtr converts a value to a pointer to that value.
// This is a generic helper function useful for creating pointers to literals.
func toPtr[T any](v T) *T {
	return &v
}

// safeIntToUint32 safely converts int to uint32, preventing integer overflow.
// This function provides secure conversion with bounds checking for production use.
func safeIntToUint32(val int) uint32 {
	if val < 0 {
		return 0
	}

	if val > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(val)
}

// deduplicateQueries removes duplicate queries to ensure consistent scoring.
// Two queries are considered duplicates if they have the same Type and Value.
// This provides defensive programming against client bugs and ensures predictable API behavior.
func deduplicateQueries(queries []*routingv1.RecordQuery) []*routingv1.RecordQuery {
	if len(queries) <= 1 {
		return queries
	}

	seen := make(map[string]bool)

	var deduplicated []*routingv1.RecordQuery

	for _, query := range queries {
		if query == nil {
			continue // Skip nil queries defensively
		}

		// Create unique key from type and value
		key := fmt.Sprintf("%s:%s", query.GetType().String(), query.GetValue())

		if !seen[key] {
			seen[key] = true

			deduplicated = append(deduplicated, query)
		}
	}

	return deduplicated
}
