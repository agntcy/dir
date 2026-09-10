// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPinHoldsEveryAgentThePackageIsInstalledInto(t *testing.T) {
	// One package is one package: a hold that covered only one agent would let a
	// bare upgrade move the others.
	seedManifest(t,
		directoryEntry("cisco.com/agent", "1.0.0", "claude-code"),
		directoryEntry("cisco.com/agent", "1.0.0", "cursor"),
		directoryEntry("cisco.com/other", "1.0.0", "claude-code"),
	)

	cmd, _ := testCmd(t)
	require.NoError(t, setPinned(cmd, "cisco.com/agent", true))

	entries := loadManifest(t).Entries
	require.Len(t, entries, 3)
	assert.True(t, entries[0].Pinned)
	assert.True(t, entries[1].Pinned)
	assert.False(t, entries[2].Pinned, "another package must not be touched")
}

func TestUnpinReleasesTheHoldWithoutMovingTheVersion(t *testing.T) {
	pinned := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	pinned.Pinned = true
	seedManifest(t, pinned)

	cmd, _ := testCmd(t)
	require.NoError(t, setPinned(cmd, "cisco.com/agent", false))

	entries := loadManifest(t).Entries
	require.Len(t, entries, 1)
	assert.False(t, entries[0].Pinned)
	assert.Equal(t, "1.0.0", entries[0].Version)
}

func TestPinningAnAlreadyPinnedPackageIsANoOp(t *testing.T) {
	pinned := directoryEntry("cisco.com/agent", "1.0.0", "claude-code")
	pinned.Pinned = true
	seedManifest(t, pinned)

	cmd, _ := testCmd(t)
	require.NoError(t, setPinned(cmd, "cisco.com/agent", true))

	assert.True(t, loadManifest(t).Entries[0].Pinned)
}

func TestPinRejectsAPackageThatIsNotInstalled(t *testing.T) {
	seedManifest(t)

	cmd, _ := testCmd(t)
	require.ErrorContains(t, setPinned(cmd, "cisco.com/nope", true), `"cisco.com/nope" is not installed`)
}

func TestPinReportWordsWhatActuallyChanged(t *testing.T) {
	assert.Equal(t, "cisco.com/agent is already pinned.", pinReport("cisco.com/agent", true, 2, 0))
	assert.Equal(t, "cisco.com/agent is already not pinned.", pinReport("cisco.com/agent", false, 2, 0))
	assert.Equal(t, "Pinned cisco.com/agent in 1 agent.", pinReport("cisco.com/agent", true, 1, 1))
	assert.Equal(t, "Pinned cisco.com/agent in 2 agents.", pinReport("cisco.com/agent", true, 2, 2))
	// One of the two agents was already held, so only one moved.
	assert.Equal(t, "Unpinned cisco.com/agent in 1 of 2 agents.",
		pinReport("cisco.com/agent", false, 2, 1))
}
