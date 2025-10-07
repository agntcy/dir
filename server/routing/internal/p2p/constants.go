// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package p2p

import "time"

// Connection Manager constants for libp2p peer connection management.
// These constants ensure healthy peer connectivity while preventing resource exhaustion.
const (
	// ConnMgrLowWater is the minimum number of connections to maintain.
	// Below this, the connection manager will not prune any peers.
	// Value accounts for: DHT routing table (~20) + GossipSub mesh (~10) + buffer (~20).
	ConnMgrLowWater = 50

	// ConnMgrHighWater is the maximum number of connections before pruning starts.
	// When this limit is reached, low-priority peers are pruned to bring count down.
	// Provides headroom for: DHT discovery + mesh dynamics + temporary connections.
	ConnMgrHighWater = 200

	// ConnMgrGracePeriod is the duration new connections are protected from pruning.
	// This gives new connections time to prove useful before being eligible for removal.
	ConnMgrGracePeriod = 2 * time.Minute
)

// Peer priority constants for Connection Manager tagging.
// Higher values indicate higher priority and are less likely to be pruned.
const (
	// PeerPriorityBootstrap is the priority for bootstrap peers.
	// Bootstrap peers are also protected (never pruned) in addition to this high priority.
	PeerPriorityBootstrap = 100

	// PeerPriorityGossipSubMesh is the priority for GossipSub mesh peers.
	// Mesh peers are critical for fast label propagation and should be kept.
	PeerPriorityGossipSubMesh = 50
)

// MeshPeerTaggingInterval defines how often GossipSub mesh peers are re-tagged
// to protect them from Connection Manager pruning as mesh topology changes.
const MeshPeerTaggingInterval = 30 * time.Second

// mDNS service name for local network peer discovery.
// This is used to identify DIR peers on the same LAN.
const MDNSServiceName = "agntcy-dir-local-discovery"
