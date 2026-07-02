// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListRunsWithoutClient(t *testing.T) {
	var out bytes.Buffer
	ListCommand.SetOut(&out)
	ListCommand.SetErr(&out)

	require.NoError(t, ListCommand.RunE(ListCommand, nil))
	require.Contains(t, out.String(), "Claude Code")
}

func TestParentHasSubcommands(t *testing.T) {
	names := map[string]bool{}
	for _, c := range Command.Commands() {
		names[c.Name()] = true
	}

	require.True(t, names["run"])
	require.True(t, names["uninstall"])
	require.True(t, names["list"])
}
