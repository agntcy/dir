// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import "time"

// DHT and routing timing constants that should be used consistently across the codebase.
// These constants ensure proper coordination between DHT expiration, republishing, and cleanup tasks.
const (
	// RecordTTL defines how long DHT records persist before expiring.
	// This is configured via dht.MaxRecordAge() and affects all PutValue operations.
	// Default DHT TTL is 36h, but we use 48h for better network resilience.
	RecordTTL = 48 * time.Hour
	// RepublishInterval defines how often we republish CID provider announcements to prevent expiration.
	// Provider records typically expire after 24h, but we use a longer interval for robustness.
	// This ensures our content remains discoverable by triggering pull-based label caching.
	RepublishInterval = 36 * time.Hour
	// CleanupInterval defines how often we clean up stale announcements.
	// This should match DHTRecordTTL to stay consistent with DHT behavior and prevent
	// our local cache from having stale entries that no longer exist in the DHT.
	CleanupInterval = 48 * time.Hour
	// RefreshInterval defines how often DHT routing tables are refreshed.
	// This is a shorter interval for maintaining network connectivity.
	RefreshInterval = 30 * time.Second
)

// Protocol constants for libp2p DHT and discovery.
const (
	// ProtocolPrefix is the prefix used for DHT protocol identification.
	ProtocolPrefix = "dir"

	// ProtocolRendezvous is the rendezvous string used for peer discovery.
	ProtocolRendezvous = "dir/connect"
)

// Validation rules and limits.
const (
	// MaxHops defines the maximum number of hops allowed in distributed queries.
	MaxHops = 20

	// NotificationChannelSize defines the buffer size for announcement notifications.
	NotificationChannelSize = 1000

	// MaxLabelAge defines when remote label announcements are considered stale.
	// Labels older than this will be cleaned up during periodic cleanup cycles.
	MaxLabelAge = 72 * time.Hour

	// DefaultMinMatchScore defines the minimum allowed match score for production safety.
	// Per proto specification: "If not set, it will return records that match at least one query".
	// Any value below this threshold is automatically corrected to this value.
	DefaultMinMatchScore = 1
)

const ResultChannelBufferSize = 100
