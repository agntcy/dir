// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"crypto/rand"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestPeerID generates a valid libp2p peer ID for use in tests.
func newTestPeerID(t *testing.T) peer.ID {
	t.Helper()

	_, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	pid, err := peer.IDFromPublicKey(pub)
	require.NoError(t, err)

	return pid
}

func TestDefaultAutosyncDisabled(t *testing.T) {
	// Deny-by-default: autosync must be off unless explicitly enabled.
	assert.False(t, DefaultAutosyncEnabled)

	var cfg AutosyncConfig
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.PeerList)
}

func TestAutosyncConfig_AllowSet_Valid(t *testing.T) {
	p1 := newTestPeerID(t)
	p2 := newTestPeerID(t)

	cfg := AutosyncConfig{
		Enabled: true,
		PeerList: []AutosyncPeer{
			{Peer: p1.String()},
			{Peer: p2.String()},
		},
	}

	allowSet, err := cfg.AllowSet()
	require.NoError(t, err)
	assert.Len(t, allowSet, 2)
	assert.Contains(t, allowSet, p1)
	assert.Contains(t, allowSet, p2)
}

func TestAutosyncConfig_AllowSet_Empty(t *testing.T) {
	cfg := AutosyncConfig{Enabled: true}

	allowSet, err := cfg.AllowSet()
	require.NoError(t, err)
	assert.Empty(t, allowSet)
}

func TestAutosyncConfig_AllowSet_Deduplicates(t *testing.T) {
	p1 := newTestPeerID(t)

	cfg := AutosyncConfig{
		PeerList: []AutosyncPeer{
			{Peer: p1.String()},
			{Peer: p1.String()},
		},
	}

	allowSet, err := cfg.AllowSet()
	require.NoError(t, err)
	assert.Len(t, allowSet, 1)
	assert.Contains(t, allowSet, p1)
}

func TestAutosyncConfig_AllowSet_InvalidPeerID(t *testing.T) {
	valid := newTestPeerID(t)

	cfg := AutosyncConfig{
		PeerList: []AutosyncPeer{
			{Peer: valid.String()},
			{Peer: "not-a-valid-peer-id"},
		},
	}

	allowSet, err := cfg.AllowSet()
	require.Error(t, err)
	assert.Nil(t, allowSet)
	// Error should identify the offending entry (index + value) for fast triage.
	assert.Contains(t, err.Error(), "peerlist[1]")
	assert.Contains(t, err.Error(), "not-a-valid-peer-id")
}
