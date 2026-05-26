// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package doctor

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapPeerMultiaddrRequiresPeerID(t *testing.T) {
	result, peerInfo := bootstrapPeerMultiaddr(0, "/dns4/bootstrap.example.com/tcp/5555")

	require.Nil(t, peerInfo)
	assert.Equal(t, "bootstrap_peer_multiaddr", result.Name)
	assert.Equal(t, statusFail, result.Status)
	assert.Contains(t, result.Message, "missing /p2p")
	assert.Equal(t, "/dns4/bootstrap.example.com/tcp/5555", result.Details["address"])
}

func TestBootstrapPeerChecksSkipNamedChecksWithoutPeers(t *testing.T) {
	results := bootstrapPeerChecks(context.Background(), validateBootstrapPeers(nil), time.Millisecond)

	require.Len(t, results, 2)
	assert.Equal(t, "bootstrap_peer_multiaddr", results[0].Name)
	assert.Equal(t, statusSkip, results[0].Status)
	assert.Equal(t, "bootstrap_peer_dial", results[1].Name)
	assert.Equal(t, statusSkip, results[1].Status)
}

func TestBootstrapPeerMultiaddrInvalid(t *testing.T) {
	result, peerInfo := bootstrapPeerMultiaddr(2, "not-a-multiaddr")

	require.Nil(t, peerInfo)
	assert.Equal(t, "bootstrap_peer_multiaddr", result.Name)
	assert.Equal(t, statusFail, result.Status)
	assert.Contains(t, result.Message, "invalid")
	assert.Equal(t, "2", result.Details["index"])
	assert.Contains(t, result.Details, "error")
}

func TestBootstrapPeerMultiaddrValid(t *testing.T) {
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	peerID, err := peerIDFromPrivateKey(priv)
	require.NoError(t, err)

	addr := "/ip4/127.0.0.1/tcp/1234/p2p/" + peerID
	result, peerInfo := bootstrapPeerMultiaddr(1, addr)

	require.NotNil(t, peerInfo)
	assert.Equal(t, "bootstrap_peer_multiaddr", result.Name)
	assert.Equal(t, statusPass, result.Status)
	assert.Equal(t, peerID, result.Details["peer_id"])
}

func TestBootstrapPeerValidationPeerInfos(t *testing.T) {
	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	peerID, err := peerIDFromPrivateKey(priv)
	require.NoError(t, err)

	validation := validateBootstrapPeers([]string{"/ip4/127.0.0.1/tcp/1234/p2p/" + peerID})

	infos := validation.peerInfos()
	require.Len(t, infos, 1)
	assert.Equal(t, peerID, infos[0].ID.String())
}

func TestBootstrapPeerChecksSkipDialForInvalidPeer(t *testing.T) {
	validation := validateBootstrapPeers([]string{"/dns4/bootstrap.example.com/tcp/5555"})

	results := bootstrapPeerChecks(context.Background(), validation, time.Millisecond)

	require.Len(t, results, 2)
	assert.Equal(t, statusFail, results[0].Status)
	assert.Equal(t, "bootstrap_peer_dial", results[1].Name)
	assert.Equal(t, statusSkip, results[1].Status)
	assert.Equal(t, "/dns4/bootstrap.example.com/tcp/5555", results[1].Details["address"])
}

func TestBootstrapPeerValidationHelpers(t *testing.T) {
	validResult := checkResult{Name: "bootstrap_peer_multiaddr", Status: statusPass}
	invalidResult := checkResult{Name: "bootstrap_peer_multiaddr", Status: statusFail}
	validation := bootstrapPeerValidation{entries: []bootstrapPeerEntry{
		{index: 0, address: "valid", result: validResult},
		{index: 1, address: "invalid", result: invalidResult},
	}}

	assert.True(t, validation.hasPeers())
	entry, ok := validation.invalidEntry()
	require.True(t, ok)
	assert.Equal(t, "invalid", entry.address)
	assert.Empty(t, validation.peerInfos())
}

func TestAddPeerProtocolDetails(t *testing.T) {
	details := map[string]string{}

	addPeerProtocolDetails(details, []protocol.ID{"/ipfs/kad/1.0.0", "/dir/rpc/0.1.0", "/meshsub/1.1.0"}, nil)

	assert.Equal(t, "3", details["protocol_count"])
	assert.Equal(t, "true", details["has_kad_dht_protocol"])
	assert.Equal(t, "true", details["has_dir_rpc_protocol"])
	assert.Equal(t, "true", details["has_gossipsub_protocol"])
	assert.True(t, hasProtocolPrefix([]string{"/dir/rpc/0.1.0"}, "/dir/rpc"))
	assert.False(t, hasProtocolPrefix([]string{"/other/1.0.0"}, "/dir/rpc"))
}

func TestAddPeerProtocolDetailsHandlesEmptyAndError(t *testing.T) {
	details := map[string]string{}
	addPeerProtocolDetails(details, nil, nil)
	assert.Equal(t, "0", details["protocol_count"])

	details = map[string]string{}
	addPeerProtocolDetails(details, nil, assert.AnError)
	assert.Equal(t, assert.AnError.Error(), details["protocol_error"])
}

func TestJoinPeerIDs(t *testing.T) {
	privA, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	privB, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	idA, err := peer.IDFromPrivateKey(privA)
	require.NoError(t, err)
	idB, err := peer.IDFromPrivateKey(privB)
	require.NoError(t, err)

	assert.Equal(t, idA.String()+","+idB.String(), joinPeerIDs([]peer.ID{idA, idB}))
	assert.Empty(t, joinPeerIDs(nil))
}

func peerIDFromPrivateKey(priv crypto.PrivKey) (string, error) {
	id, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", fmt.Errorf("create peer ID from private key: %w", err)
	}

	return id.String(), nil
}
