// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

var (
	DefaultListenAddress  = "/ip4/0.0.0.0/tcp/8999"
	DefaultBootstrapPeers = []string{
		// TODO: once we deploy our bootstrap nodes, we should update this
	}
)

type Config struct {
	// Address to use for routing
	ListenAddress string `json:"listen_address,omitempty" mapstructure:"listen_address"`

	// Peers to use for bootstrapping.
	// We can choose between public and private peers.
	BootstrapPeers []string `json:"bootstrap_peers,omitempty" mapstructure:"bootstrap_peers"`

	// Path to asymmetric private key
	KeyPath string `json:"key_path,omitempty" mapstructure:"key_path"`

	// Path to the routing datastore.
	// If empty, the routing data will be stored in memory.
	// If not empty, this dir will be used to store the routing data on disk.
	DatastoreDir string `json:"datastore_dir,omitempty" mapstructure:"datastore_dir"`
}
