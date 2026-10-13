// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package conv converts integers to a narrower unsigned type.
// Uint32* returns an error when n is out of range. SafeUint32*
// clamps negatives to 0 and overflow to math.MaxUint32.
package conv

import (
	"fmt"
	"math"
)

// Uint32 converts n to uint32. Negative values and values above
// math.MaxUint32 return an error.
func Uint32(n int) (uint32, error) {
	return Uint32FromInt64(int64(n))
}

// Uint32FromInt64 converts n to uint32. Negative values and values above
// math.MaxUint32 return an error.
func Uint32FromInt64(n int64) (uint32, error) {
	if n < 0 || n > math.MaxUint32 {
		return 0, fmt.Errorf("can't convert %d to uint32", n)
	}

	return uint32(n), nil
}

// SafeUint32 converts n to uint32, clamping out-of-range values.
func SafeUint32(n int) uint32 {
	return SafeUint32FromInt64(int64(n))
}

// SafeUint32FromInt64 converts n to uint32. Negative values become 0;
// values above math.MaxUint32 become math.MaxUint32.
func SafeUint32FromInt64(n int64) uint32 {
	if n < 0 {
		return 0
	}

	if n > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(n)
}
