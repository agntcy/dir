// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import "math"

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
