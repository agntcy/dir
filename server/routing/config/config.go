// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"time"
)

var (
	DefaultListenAddress  = "/ip4/0.0.0.0/tcp/8999"
	DefaultBootstrapPeers = []string{
		// TODO: once we deploy our bootstrap nodes, we should update this
	}

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
	// If not empty, this dir will be used to persist the DHT's state on disk:
	// its routing table and the provider records it holds for other peers.
	DatastoreDir string `json:"datastore_dir,omitempty" mapstructure:"datastore_dir"`

	// Refresh interval for DHT routing tables.
	// If not set or zero, uses the default RefreshInterval constant.
	// This is primarily used for testing with faster intervals.
	RefreshInterval time.Duration `json:"refresh_interval,omitempty" mapstructure:"refresh_interval"`

	// RepublishInterval controls how often the records this node has published
	// are re-advertised, by CID and by label, so their provider records do not
	// expire. They are also advertised once at startup regardless.
	// If not set or zero, uses the default RepublishInterval constant (36h).
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
}
