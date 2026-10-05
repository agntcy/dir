// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package conv

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUint32(t *testing.T) {
	t.Parallel()

	got, err := Uint32(0)
	require.NoError(t, err)
	require.Equal(t, uint32(0), got)

	got, err = Uint32(math.MaxInt32)
	require.NoError(t, err)
	require.Equal(t, uint32(math.MaxInt32), got)

	_, err = Uint32(-1)
	require.EqualError(t, err, "can't convert -1 to uint32")
}

func TestUint32FromInt64(t *testing.T) {
	t.Parallel()

	got, err := Uint32FromInt64(0)
	require.NoError(t, err)
	require.Equal(t, uint32(0), got)

	got, err = Uint32FromInt64(math.MaxUint32)
	require.NoError(t, err)
	require.Equal(t, uint32(math.MaxUint32), got)

	_, err = Uint32FromInt64(-1)
	require.EqualError(t, err, "can't convert -1 to uint32")

	_, err = Uint32FromInt64(int64(math.MaxUint32) + 1)
	require.EqualError(t, err, "can't convert 4294967296 to uint32")
}

func TestSafeUint32(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint32(0), SafeUint32(0))
	require.Equal(t, uint32(0), SafeUint32(-1))
	require.Equal(t, uint32(math.MaxInt32), SafeUint32(math.MaxInt32))
}

func TestSafeUint32FromInt64(t *testing.T) {
	t.Parallel()

	require.Equal(t, uint32(0), SafeUint32FromInt64(0))
	require.Equal(t, uint32(0), SafeUint32FromInt64(-1))
	require.Equal(t, uint32(math.MaxUint32), SafeUint32FromInt64(math.MaxUint32))
	require.Equal(t, uint32(math.MaxUint32), SafeUint32FromInt64(int64(math.MaxUint32)+1))
}
