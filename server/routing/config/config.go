// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

var (
	DefaultListenAddress  = "/ip4/0.0.0.0/tcp/8999"
	DefaultBootstrapPeers = []string{
		// TODO: once we deploy our bootstrap nodes, we should update this
	}

	// GossipSub default (only enable/disable is configurable).
	DefaultGossipSubEnabled = true

	// Autosync default (disabled by default; deny-by-default policy).
	DefaultAutosyncEnabled = false

	// RelayService default (disabled; enable only on publicly-reachable nodes).
	DefaultRelayServiceEnabled = false

	// ForceReachabilityPrivate default (disabled; let AutoNAT decide reachability).
	DefaultForceReachabilityPrivate = false

	// ForceReachabilityPublic default (disabled; let AutoNAT decide reachability).
	DefaultForceReachabilityPublic = false
)

type Config struct {
	// Address to use for routing
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// Address to use for sync operations
	DirectoryAPIAddress string `json:"directory_api_address,omitempty" mapstructure:"directory_api_address"`

	// DirectoryOCIAddress is this node's OCI registry endpoint advertised to
	// peers (as an "/oci/<addr>" host multiaddr) so remote peers can pull record
	// content directly from this node's registry. Optional; empty means not
	// advertised. Independent of store.oci.registry_address — set the publicly
	// reachable registry endpoint here.
	DirectoryOCIAddress string `json:"directory_oci_address,omitempty" mapstructure:"directory_oci_address"`

	// Peers to use for bootstrapping.
	// We can choose between public and private peers.
	BootstrapPeers []string `json:"bootstrap_peers,omitempty" mapstructure:"bootstrap_peers"`

	// Path to asymmetric private key
	KeyPath string `json:"key_path,omitempty" mapstructure:"key_path"`

	// Path to the routing datastore.
	// If empty, the routing data will be stored in memory.
	// If not empty, this dir will be used to store the routing data on disk.
	DatastoreDir string `json:"datastore_dir,omitempty" mapstructure:"datastore_dir"`

	// Refresh interval for DHT routing tables.
	// If not set or zero, uses the default RefreshInterval constant.
	// This is primarily used for testing with faster intervals.
	RefreshInterval time.Duration `json:"refresh_interval,omitempty" mapstructure:"refresh_interval"`

	// RepublishInterval controls how often local CID provider announcements are
	// republished to keep content discoverable (DHT provider records + GossipSub
	// labels). If not set or zero, uses the default RepublishInterval constant (36h).
	// Lower values let newly joined nodes converge on existing content sooner, at
	// the cost of more frequent announcement traffic.
	RepublishInterval time.Duration `json:"republish_interval,omitempty" mapstructure:"republish_interval"`

	// RelayService enables a circuit-relay v2 service on this node so it can
	// relay traffic for NAT'd peers. Enable only on publicly-reachable nodes
	// (e.g. bootstrap nodes); it consumes bandwidth on behalf of other peers.
	RelayService bool `json:"relay_service,omitempty" mapstructure:"relay_service"`

	// StaticRelays is a list of relay multiaddrs (each including /p2p/<peer-id>)
	// this node uses as AutoRelay static relays to obtain circuit addresses when
	// it is behind NAT. Configured via config file/YAML only (list of strings).
	StaticRelays []string `json:"static_relays,omitempty" mapstructure:"static_relays"`

	// ForceReachabilityPrivate makes this node assume it is not publicly
	// reachable, so AutoRelay proactively reserves a relay and advertises a
	// circuit address (direct dials + DCUtR hole punching are still preferred).
	// Enable only on nodes known to be behind NAT; leave false to let AutoNAT
	// decide. Has no effect on genuinely public nodes if left false.
	ForceReachabilityPrivate bool `json:"force_reachability_private,omitempty" mapstructure:"force_reachability_private"`

	// ForceReachabilityPublic makes this node assume it is publicly reachable.
	// This is REQUIRED for a relay node (RelayService: true) that sits behind a
	// cloud load balancer: the circuit-relay v2 hop service only starts once the
	// host's reachability is Public, and AutoNAT cannot self-confirm reachability
	// behind an LB, so it would otherwise stay Unknown and never serve relay
	// reservations. Enable only on genuinely public nodes. Mutually exclusive
	// with ForceReachabilityPrivate.
	ForceReachabilityPublic bool `json:"force_reachability_public,omitempty" mapstructure:"force_reachability_public"`

	// GossipSub configuration for label announcements
	GossipSub GossipSubConfig `json:"gossipsub" mapstructure:"gossipsub"`

	// Autosync configuration for DHT-based record + referrer synchronization
	Autosync AutosyncConfig `json:"autosync" mapstructure:"autosync"`
}

// GossipSubConfig configures GossipSub-based label announcements.
// Protocol parameters (topic name, message size limits) are NOT configurable
// and are defined in server/routing/pubsub/constants.go to ensure network-wide
// compatibility. Only the enable/disable flag is configurable.
//
// Benefits when enabled:
//   - Reaches ALL subscribed peers (not just k-closest in DHT)
//   - Minimal bandwidth (~100B vs KB-MB for full record)
//   - Fast propagation (~5-20ms vs ~100-500ms for DHT)
//   - High cache hit rate (90%+ vs 30% with pull-based)
type GossipSubConfig struct {
	// Enabled controls whether GossipSub label announcements are used.
	// When true: Labels are announced via GossipSub (efficient, wide propagation)
	// When false: Falls back to DHT+Pull mechanism (existing behavior)
	// Default: true (recommended for production)
	//
	// Note: Protocol parameters (topic, message size) are hardcoded in
	// server/routing/pubsub/constants.go for network compatibility.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`
}

// AutosyncConfig configures DHT-based record + referrer synchronization.
//
// When enabled, DHT provider announcements originating from a peer in PeerList
// trigger the node to pull the announced record (and its referrers) from that
// peer over libp2p RPC and ingest them locally with full parity to a normal
// push (content store + search index + referrer state).
//
// The policy is deny-by-default: only peers explicitly listed in PeerList are
// ever synced from. Allow-list matching is performed against libp2p's
// authenticated peer.ID (never a self-reported/payload field).
type AutosyncConfig struct {
	// Enabled controls whether DHT-based autosync is active.
	// Default: false (deny-by-default).
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// PeerList is the allow-list of trusted source peers to auto-sync from.
	//
	// Note: this is a list of objects (not bare peer-ID strings) so that
	// per-peer policy fields (e.g. a future "republish" flag) can be added
	// without a breaking config change. Because it is a list of structs, it is
	// configured via config file/YAML only (not via a single environment
	// variable).
	PeerList []AutosyncPeer `json:"peerlist,omitempty" mapstructure:"peerlist"`
}

// AutosyncPeer identifies a single trusted source peer in the autosync
// allow-list.
type AutosyncPeer struct {
	// Peer is the libp2p peer ID of the trusted source peer
	// (e.g. "12D3KooW...").
	Peer string `json:"peer" mapstructure:"peer"`

	// NOTE: A "Republish" flag is intentionally deferred to a future iteration.
	// Keeping this a struct (rather than a bare string) makes adding it later a
	// non-breaking change.
}

// AllowSet parses the configured PeerList into a set of libp2p peer IDs for
// O(1) membership checks by the autosync manager.
//
// It fails fast: an invalid peer ID returns an error identifying the offending
// entry, rather than being silently skipped. This is deliberate for a security
// allow-list — a typo in a trusted peer ID should be surfaced at startup, not
// silently ignored (which could otherwise cause a trusted peer to never sync).
func (c AutosyncConfig) AllowSet() (map[peer.ID]struct{}, error) {
	allowSet := make(map[peer.ID]struct{}, len(c.PeerList))

	for i, p := range c.PeerList {
		pid, err := peer.Decode(p.Peer)
		if err != nil {
			return nil, fmt.Errorf("invalid autosync peer ID at peerlist[%d] (%q): %w", i, p.Peer, err)
		}

		allowSet[pid] = struct{}{}
	}

	return allowSet, nil
}
