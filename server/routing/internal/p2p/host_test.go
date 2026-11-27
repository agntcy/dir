// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"crypto/rand"
	"slices"
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/stretchr/testify/require"
)

func newTestHost(t *testing.T, opts options) host.Host {
	t.Helper()

	key, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)

	opts.Key = key
	opts.ListenAddress = "/ip4/127.0.0.1/tcp/0"

	h, err := newHost(&opts)
	require.NoError(t, err)

	t.Cleanup(func() { _ = h.Close() })

	return h
}

func TestNewHost(t *testing.T) {
	hasLoopback := func(addrs []ma.Multiaddr) bool {
		return slices.ContainsFunc(addrs, manet.IsIPLoopback)
	}

	hasDirAddr := func(addrs []ma.Multiaddr) bool {
		for _, a := range addrs {
			if _, err := a.ValueForProtocol(DirProtocolCode); err == nil {
				return true
			}
		}

		return false
	}

	t.Run("private IPs allowed", func(t *testing.T) {
		h := newTestHost(t, options{DirectoryAPIAddress: "dir.example.com:8888"})

		require.True(t, hasLoopback(h.Addrs()))
		require.True(t, hasDirAddr(h.Addrs()))
	})

	t.Run("private IPs disabled", func(t *testing.T) {
		h := newTestHost(t, options{DirectoryAPIAddress: "dir.example.com:8888", DisablePrivateIPs: true})

		require.False(t, hasLoopback(h.Addrs()))
		require.True(t, hasDirAddr(h.Addrs()))
	})

	t.Run("both reachability flags set", func(t *testing.T) {
		newTestHost(t, options{ForceReachabilityPublic: true, ForceReachabilityPrivate: true})
	})
}

func TestSetupMDNSDisabled(t *testing.T) {
	// mDNS exposes nothing to inspect, so this only checks the early return
	// does not start the service (which would log and bind a multicast socket).
	setupMDNS(newTestHost(t, options{}), true)
}
