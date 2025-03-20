// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package config

var (
	DefaultAddress        = "/ip4/0.0.0.0/tcp/0"
	DefaultBootstrapPeers = []string{
		// TODO: once we deploy our bootstrap nodes, we should update this
	}
)

type Config struct {
	// Expose routing to public network
	Public bool `json:"local_dir,omitempty" mapstructure:"local_dir"`

	// Address to use for routing
	Address string `json:"address,omitempty" mapstructure:"address"`

	// Peers to use for bootstrapping.
	// We can choose between public and private peers.
	BootstrapPeers []string `json:"bootstrap_peers,omitempty" mapstructure:"bootstrap_peers"`
}
