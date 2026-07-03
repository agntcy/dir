// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveChosenAllMeansEmpty(t *testing.T) {
	chosen, err := resolveChosen([]string{"all"})
	require.NoError(t, err)
	require.Empty(t, chosen, `"all" resolves to an empty set (all detected)`)
}

func TestResolveChosenExplicitList(t *testing.T) {
	chosen, err := resolveChosen([]string{"claude-code", "cursor"})
	require.NoError(t, err)
	require.Equal(t, map[string]bool{"claude-code": true, "cursor": true}, chosen)
}

func TestResolveChosenUnknownAgentErrors(t *testing.T) {
	_, err := resolveChosen([]string{"cusror"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown agent")
}

func TestResolveChosenAllCombinedWithSpecificErrors(t *testing.T) {
	_, err := resolveChosen([]string{"all", "cursor"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot be combined")
}
