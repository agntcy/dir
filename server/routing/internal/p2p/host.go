// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"fmt"
	"net/url"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	connmgr "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	ma "github.com/multiformats/go-multiaddr"
)

// EncodeAppAddr percent-encodes an application address (e.g. "ghcr.io/org/repo")
// into a single multiaddr path segment, escaping "/" so it does not collide with
// the multiaddr component separator. This mirrors the multiaddr /http-path
// convention (RFC 3986 segment-nz encoding); the binary form is length-prefixed
// so no encoding is needed there.
func EncodeAppAddr(addr string) string {
	return url.PathEscape(addr)
}

// DecodeAppAddr reverses EncodeAppAddr. On malformed input it returns the value
// unchanged (best effort) so callers always get a usable string.
func DecodeAppAddr(seg string) string {
	decoded, err := url.PathUnescape(seg)
	if err != nil {
		return seg
	}

	return decoded
}

// buildAppAddrs constructs the advertised /dir/ and /oci/ host multiaddrs from
// the configured (URL-form) addresses, skipping empty or invalid values.
func buildAppAddrs(dirAPIAddr, ociAddr string) []ma.Multiaddr {
	var out []ma.Multiaddr

	if a := newAppAddr(DirProtocol, dirAPIAddr); a != nil {
		out = append(out, a)
	}

	if a := newAppAddr(OciProtocol, ociAddr); a != nil {
		out = append(out, a)
	}

	return out
}

// newAppAddr builds a "/<proto>/<percent-encoded addr>" multiaddr, or returns
// nil (logging once) when the address is empty or cannot be represented.
func newAppAddr(proto, addr string) ma.Multiaddr {
	if addr == "" {
		return nil
	}

	m, err := ma.NewMultiaddr("/" + proto + "/" + EncodeAppAddr(addr))
	if err != nil {
		logger.Warn("Ignoring invalid routing address; not advertised",
			"protocol", proto, "value", addr, "error", err)

		return nil
	}

	return m
}

const (
	DirProtocol     = "dir"
	DirProtocolCode = 65535
	OciProtocol     = "oci"
	OciProtocolCode = 65534
)

// Register the custom /dir/ and /oci/ multiaddr protocols so a node's Directory
// API and OCI registry endpoints can be carried as host multiaddrs and
// propagated to other peers via libp2p identify.
func init() {
	if err := addStringProtocol(DirProtocol, DirProtocolCode); err != nil {
		panic(err)
	}

	if err := addStringProtocol(OciProtocol, OciProtocolCode); err != nil {
		panic(err)
	}
}

// addStringProtocol registers a length-prefixed string multiaddr protocol,
// letting an application-level address (e.g. host:port) ride along as a host
// multiaddr component such as "/dir/<addr>" or "/oci/<addr>".
func addStringProtocol(name string, code int) error {
	err := ma.AddProtocol(ma.Protocol{
		Name:  name,
		Code:  code,
		VCode: ma.CodeToVarint(code),
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
		return fmt.Errorf("failed to add %q multiaddr protocol: %w", name, err)
	}

	return nil
}

// newHost creates a new host libp2p host.
func newHost(listenAddr, dirAPIAddr, ociAddr string, key crypto.PrivKey, enableRelayService, forceReachabilityPrivate, forceReachabilityPublic bool) (host.Host, error) {
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

	// Precompute the advertised Directory API (/dir/) and OCI registry (/oci/)
	// multiaddrs once; the AddrsFactory below runs on every address query. Values
	// are percent-encoded so URL-form addresses (e.g. "ghcr.io/org/repo") survive
	// the multiaddr separator. Invalid values are logged once and skipped.
	appAddrs := buildAppAddrs(dirAPIAddr, ociAddr)

	hostOpts := []libp2p.Option{
		// Advertise the app-level endpoints as host multiaddrs so peers learn
		// them via identify.
		libp2p.AddrsFactory(
			func(addrs []ma.Multiaddr) []ma.Multiaddr {
				return append(addrs, appAddrs...)
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
