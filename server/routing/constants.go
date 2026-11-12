// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import "time"

// Default DHT and routing timing settings. Each one can be overridden through the
// matching routing config key (see server/routing/config/config.go).
// These constants ensure proper coordination between DHT expiration, republishing, and cleanup tasks.
const (
	// RecordTTL defines how long DHT provider records persist before expiring.
	// It is applied to both dht.MaxRecordAge() and the provider store validity.
	RecordTTL = 48 * time.Hour
	// RepublishInterval defines how often we republish CID provider announcements to prevent expiration.
	// It is set to RecordTTL/2 so records are refreshed well before they expire.
	RepublishInterval = 24 * time.Hour
	// CleanupInterval defines how often we clean up stale announcements.
	// This should match DHTRecordTTL to stay consistent with DHT behavior and prevent
	// our local cache from having stale entries that no longer exist in the DHT.
	CleanupInterval = 48 * time.Hour
	// RefreshInterval defines how often DHT routing tables are refreshed.
	RefreshInterval = 10 * time.Minute
)

// Default DHT lookup parameters, tuned for a small and stable network.
// Overridable through routing.bucket_size and routing.concurrency.
// BucketSize should be the same on every node of the network.
const (
	// BucketSize is the Kademlia k parameter (default 20).
	BucketSize = 10

	// Concurrency is the Kademlia alpha parameter (default 10).
	Concurrency = 5
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
	MaxLabelAge = 48 * time.Hour

	// DefaultMinMatchScore defines the minimum allowed match score for production safety.
	// Per proto specification: "If not set, it will return records that match at least one query".
	// Any value below this threshold is automatically corrected to this value.
	DefaultMinMatchScore = 1
)

const ResultChannelBufferSize = 100
