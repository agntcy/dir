// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package safefetch

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadLimited_EnforcesMaxBytes(t *testing.T) {
	body, err := readLimited(strings.NewReader(strings.Repeat("a", 100)), 10)
	require.Error(t, err)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestReadLimited_AllowsExactMax(t *testing.T) {
	body, err := readLimited(strings.NewReader(strings.Repeat("a", 10)), 10)
	require.NoError(t, err)
	assert.Len(t, body, 10)
}
