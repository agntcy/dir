// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

// Routing defaults.
var (
	// DefaultRoutingListenAddress is the default libp2p listen
	// multiaddr for the routing subsystem.
	DefaultRoutingListenAddress = "/ip4/0.0.0.0/tcp/8999"

	// DefaultBootstrapPeers is the default set of bootstrap peers.
	// Currently empty; once dir bootstrap nodes exist, populate here.
	DefaultBootstrapPeers = []string{}

	// DefaultGossipSubEnabled enables GossipSub label announcements
	// by default.
	DefaultGossipSubEnabled = true
)

// Routing holds the libp2p / DHT routing configuration used by the
// apiserver and the daemon (embedded mode).
type Routing struct {
	// ListenAddress is the libp2p listen multiaddr.
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// DirectoryAPIAddress is the address advertised for sync.
	DirectoryAPIAddress string `json:"directory_api_address,omitempty" mapstructure:"directory_api_address"`

	// BootstrapPeers lists peers used for DHT bootstrap.
	BootstrapPeers []string `json:"bootstrap_peers,omitempty" mapstructure:"bootstrap_peers"`

	// KeyPath is the path to the libp2p private key file.
	KeyPath string `json:"key_path,omitempty" mapstructure:"key_path"`

	// DatastoreDir is the on-disk routing datastore. Empty means
	// in-memory.
	DatastoreDir string `json:"datastore_dir,omitempty" mapstructure:"datastore_dir"`

	// RefreshInterval is the DHT routing-table refresh interval.
	// Zero means use the libp2p default; primarily for tests.
	RefreshInterval time.Duration `json:"refresh_interval,omitempty" mapstructure:"refresh_interval"`

	// GossipSub holds GossipSub configuration for label announcements.
	GossipSub GossipSub `json:"gossipsub" mapstructure:"gossipsub"`
}

// GossipSub configures GossipSub-based label announcements. Protocol
// parameters (topic name, message size limits) are intentionally not
// configurable; see server/routing/pubsub/constants.go.
//
// Benefits when enabled:
//   - Reaches ALL subscribed peers (not just k-closest in DHT)
//   - Minimal bandwidth (~100B vs KB-MB for full record)
//   - Fast propagation (~5-20ms vs ~100-500ms for DHT)
//   - High cache hit rate (90%+ vs 30% with pull-based)
type GossipSub struct {
	// Enabled controls whether GossipSub label announcements are used.
	// When true, labels are announced via GossipSub. When false, falls
	// back to the DHT+Pull mechanism.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`
}
