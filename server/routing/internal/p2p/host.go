// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"fmt"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"
)

const (
	DirProtocol     = "dir"
	DirProtocolCode = 65535
)

// Add dir protocol to the host.
//
//nolint:mnd
func init() {
	err := ma.AddProtocol(ma.Protocol{
		Name:  DirProtocol,
		Code:  DirProtocolCode,
		VCode: ma.CodeToVarint(DirProtocolCode),
		Size:  ma.LengthPrefixedVarSize,
		Transcoder: ma.NewTranscoderFromFunctions(
			// String to bytes encoder
			func(s string) ([]byte, error) {
				return []byte(s), nil
			},
			// Bytes to string decoder
			func(b []byte) (string, error) {
				return string(b), nil
			},
			// Validator (optional)
			nil,
		),
	})
	if err != nil {
		panic(fmt.Errorf("failed to add dir protocol: %w", err))
	}
}

// newHost creates a new host libp2p host.
func newHost(listenAddr, dirAPIAddr string, key crypto.PrivKey, enableRelayService, forceReachabilityPrivate, forceReachabilityPublic bool) (host.Host, error) {
	// Create connection manager to limit and manage peer connections.
	// This prevents resource exhaustion and enables smart peer pruning based on priority.
	connMgr, err := connmgr.NewConnManager(
		ConnMgrLowWater,  // Minimum connections (DHT + GossipSub + buffer)
		ConnMgrHighWater, // Maximum connections (prevents resource exhaustion)
		connmgr.WithGracePeriod(ConnMgrGracePeriod), // Protect new connections
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host connection manager: %w", err)
	}

	hostOpts := []libp2p.Option{
		// Add directory API address to the host address factory
		libp2p.AddrsFactory(
			func(addrs []ma.Multiaddr) []ma.Multiaddr {
				// Only add the dir address if dirAPIAddr is not empty
				if dirAPIAddr != "" {
					dirAddr := ma.StringCast("/dir/" + dirAPIAddr)

					return append(addrs, dirAddr)
				}

				return addrs
			},
		),
		// Use the keypair we generated
		libp2p.Identity(key),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(listenAddr),
		// support TLS connections
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		// support any other default transports (TCP)
		libp2p.DefaultTransports,
		// support any other default multiplexer
		libp2p.DefaultMuxers,
		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(connMgr),
		// Enable hole punching to upgrade relay connections to direct.
		// When two NAT'd peers connect via relay, hole punching attempts to
		// establish a direct connection through simultaneous dialing (DCUtR protocol).
		// Success rate: ~70-80%. Falls back to relay if hole punching fails.
		libp2p.EnableHolePunching(),
		// Attempt to open ports using uPNP for NATed hosts.
		libp2p.NATPortMap(),
		// Enable AutoNAT service to help other peers detect if they are behind NAT.
		// This is the server-side component that responds to NAT detection requests.
		// Note: AutoNAT client (for detecting our own NAT status) runs automatically.
		// This service is highly rate-limited and should not cause any performance issues.
		libp2p.EnableNATService(),
	}

	// Enable a circuit-relay v2 service on publicly-reachable nodes so they can
	// relay traffic for NAT'd peers (and let DCUtR coordinate hole punching).
	if enableRelayService {
		hostOpts = append(hostOpts, libp2p.EnableRelayService())
	}

	// Reachability overrides (mutually exclusive). Public wins if both are set.
	//
	// ForceReachabilityPublic is required for a relay node behind a cloud load
	// balancer: the circuit-relay v2 hop service (EnableRelayService) only starts
	// when the host's reachability is Public, and AutoNAT cannot self-confirm
	// reachability behind an LB, so it would otherwise stay Unknown and never
	// serve reservations.
	//
	// ForceReachabilityPrivate makes a NAT'd node proactively reserve a relay and
	// advertise a circuit address (direct dials + DCUtR are still preferred).
	switch {
	case forceReachabilityPublic:
		if forceReachabilityPrivate {
			logger.Warn("Both force_reachability_public and force_reachability_private are set; using public")
		}

		hostOpts = append(hostOpts, libp2p.ForceReachabilityPublic())
	case forceReachabilityPrivate:
		hostOpts = append(hostOpts, libp2p.ForceReachabilityPrivate())
	}

	host, err := libp2p.New(hostOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create p2p host: %w", err)
	}

	return host, nil
}
