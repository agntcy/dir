// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agntcy/dir/utils/env"
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

	// DisablePrivateIPs default (private addresses are allowed).
	DefaultDisablePrivateIPs = false
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
	// If not set or zero, uses the default RefreshInterval constant (10m).
	RefreshInterval time.Duration `json:"refresh_interval,omitempty" mapstructure:"refresh_interval"`

	// RecordTTL is how long DHT provider records are kept before they expire.
	// If not set or zero, uses the default RecordTTL constant (48h).
	// Must be greater than DHTReprovideInterval.
	RecordTTL time.Duration `json:"record_ttl,omitempty" mapstructure:"record_ttl"`

	// CleanupInterval is how often stale remote labels are removed.
	// If not set or zero, uses the default CleanupInterval constant (48h).
	CleanupInterval time.Duration `json:"cleanup_interval,omitempty" mapstructure:"cleanup_interval"`

	// MaxLabelAge is the age after which a remote label is considered stale.
	// If not set or zero, uses the default MaxLabelAge constant (48h).
	// Must be greater than DHTReprovideInterval so labels are refreshed before they go stale.
	MaxLabelAge time.Duration `json:"max_label_age,omitempty" mapstructure:"max_label_age"`

	// BucketSize is the Kademlia k parameter.
	// If not set or zero, uses the default BucketSize constant (10).
	// Should be the same on every node of the network.
	BucketSize int `json:"bucket_size,omitempty" mapstructure:"bucket_size"`

	// Concurrency is the Kademlia alpha parameter (parallel lookup requests).
	// If not set or zero, uses the default Concurrency constant (5).
	Concurrency int `json:"concurrency,omitempty" mapstructure:"concurrency"`

	// DHTReprovideInterval controls how often local CID provider records are
	// re-announced to the DHT so they do not expire on remote peers.
	// If not set or zero, uses the default DHTReprovideInterval constant (24h).
	// Must be lower than RecordTTL.
	DHTReprovideInterval time.Duration `json:"dht_reprovide_interval,omitempty" mapstructure:"dht_reprovide_interval"`

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

	// DisablePrivateIPs restricts the DHT and host to public IP addresses
	// (drops private, loopback and unspecified ones) and disables mDNS LAN
	// discovery. Enable on internet-facing nodes to avoid dialing private
	// addresses; leave false for private networks (Kubernetes, LAN, local dev).
	DisablePrivateIPs bool `json:"disable_private_ips,omitempty" mapstructure:"disable_private_ips"`

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

	// ReprovideInterval controls how often local records are re-announced over
	// GossipSub, independently of the DHT. Lower values let newly joined nodes
	// learn labels sooner, at the cost of more announcement traffic.
	// If not set or zero, uses the default GossipReprovideInterval constant (24h).
	ReprovideInterval time.Duration `json:"reprovide_interval,omitempty" mapstructure:"reprovide_interval"`
}

// removedKeys maps routing keys that no longer exist to their replacement, so
// a stale configuration fails loudly instead of being silently ignored.
var removedKeys = map[string][]string{
	"republish_interval": {"dht_reprovide_interval", "gossipsub.reprovide_interval"},
}

// CheckRemovedKeys returns an error if a removed routing key is still set in
// the configuration file or through its environment variable. configFile is
// named in the error when the key comes from it, keyPrefix is the path of the
// routing block (e.g. "routing." or "server.routing.") and envPrefix the
// loader's variable prefix.
func CheckRemovedKeys(inConfig func(key string) bool, configFile, keyPrefix, envPrefix string) error {
	for key, replacements := range removedKeys {
		full := keyPrefix + key

		name := env.Name(envPrefix, full)
		_, inEnv := os.LookupEnv(name)

		var source string

		switch {
		case inConfig(full) && configFile != "":
			source = configFile
		case inConfig(full):
			source = "the configuration file"
		case inEnv:
			source = name
		default:
			continue
		}

		use := make([]string, len(replacements))
		for i, r := range replacements {
			use[i] = keyPrefix + r
		}

		return fmt.Errorf("%s (set in %s) was removed, use %s instead", full, source, strings.Join(use, " and "))
	}

	return nil
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
