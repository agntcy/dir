// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

var (
	DefaultListenAddress  = "/ip4/0.0.0.0/tcp/8999"
	DefaultBootstrapPeers = []string{
		// TODO: once we deploy our bootstrap nodes, we should update this
	}

	// GossipSub default (only enable/disable is configurable).
	DefaultGossipSubEnabled = true
)

type Config struct {
	// Address to use for routing
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// Address to use for sync operations
	DirectoryAPIAddress string `json:"directory_api_address,omitempty" mapstructure:"directory_api_address"`

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

	// GossipSub configuration for label announcements
	GossipSub GossipSubConfig `json:"gossipsub,omitempty" mapstructure:"gossipsub"`
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
