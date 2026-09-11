// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import "time"

// Connection Manager constants for libp2p peer connection management.
// These constants ensure healthy peer connectivity while preventing resource exhaustion.
const (
	// ConnMgrLowWater is the minimum number of connections to maintain.
	// Below this, the connection manager will not prune any peers.
	// Value accounts for: DHT routing table (~20) + buffer.
	ConnMgrLowWater = 50

	// ConnMgrHighWater is the maximum number of connections before pruning starts.
	// When this limit is reached, low-priority peers are pruned to bring count down.
	// Provides headroom for: DHT discovery + mesh dynamics + temporary connections.
	ConnMgrHighWater = 200

	// ConnMgrGracePeriod is the duration new connections are protected from pruning.
	// This gives new connections time to prove useful before being eligible for removal.
	ConnMgrGracePeriod = 2 * time.Minute
)

// PeerPriorityBootstrap is the Connection Manager priority for bootstrap peers.
// Higher values are less likely to be pruned; bootstrap peers are also
// protected outright.
const PeerPriorityBootstrap = 100

// mDNS service name for local network peer discovery.
// This is used to identify DIR peers on the same LAN.
const MDNSServiceName = "agntcy-dir-local-discovery"
