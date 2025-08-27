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
	// RepublishInterval defines how often we republish DHT records to prevent expiration.
	// This should be significantly less than DHTRecordTTL to ensure records don't expire.
	// We use 36h (75% of DHTRecordTTL) to provide a safe margin for network delays.
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
)

// AnnouncementType defines the type of DHT announcement being processed.
// This helps distinguish between different kinds of network events and route them appropriately.
type AnnouncementType string

const (
	// AnnouncementTypeCID indicates a content identifier provider announcement.
	// This means "I have this content" - peers announce their ability to serve specific records.
	AnnouncementTypeCID AnnouncementType = "CID"

	// AnnouncementTypeLabel indicates a semantic label mapping announcement.
	// This means "this content has these labels" - peers announce skill/domain/feature associations.
	AnnouncementTypeLabel AnnouncementType = "LABEL"
)

const ResultChannelBufferSize = 100
