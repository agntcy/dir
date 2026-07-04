// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"strings"
	"time"
)

var (
	DefaultRoutingListenAddress = "/ip4/0.0.0.0/tcp/8999"
	DefaultBootstrapPeers       = []string{}
	DefaultGossipSubEnabled     = true
)

// Routing configures the libp2p routing layer.
type Routing struct {
	ListenAddress       string        `json:"listen_address,omitempty"        mapstructure:"listen_address"`
	DirectoryAPIAddress string        `json:"directory_api_address,omitempty" mapstructure:"directory_api_address"`
	BootstrapPeers      []string      `json:"bootstrap_peers,omitempty"       mapstructure:"bootstrap_peers"`
	KeyPath             string        `json:"key_path,omitempty"              mapstructure:"key_path"`
	DatastoreDir        string        `json:"datastore_dir,omitempty"         mapstructure:"datastore_dir"`
	RefreshInterval     time.Duration `json:"refresh_interval,omitempty"      mapstructure:"refresh_interval"`
	GossipSub           GossipSub     `json:"gossipsub"                       mapstructure:"gossipsub"`
}

// GossipSub configures GossipSub-based label announcements.
type GossipSub struct {
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`
}

// defaultBootstrapPeersStr returns the bootstrap peers as a comma-separated
// string suitable for a viper default value.
func defaultBootstrapPeersStr() string {
	return strings.Join(DefaultBootstrapPeers, ",")
}
