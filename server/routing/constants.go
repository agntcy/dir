// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import "time"

// DHT and routing timing constants that should be used consistently across the codebase.
// These constants ensure proper coordination between DHT expiration, republishing, and cleanup tasks.
const (
	// DHTRecordTTL defines how long DHT records persist before expiring.
	// This is configured via dht.MaxRecordAge() and affects all PutValue operations.
	// Default DHT TTL is 36h, but we use 48h for better network resilience.
	DHTRecordTTL = 48 * time.Hour

	// LabelRepublishInterval defines how often we republish local label mappings to prevent DHT expiration.
	// This should be significantly less than DHTRecordTTL to ensure records don't expire.
	// We use 36h (75% of DHTRecordTTL) to provide a safe margin for network delays.
	LabelRepublishInterval = 36 * time.Hour

	// RemoteLabelCleanupInterval defines how often we clean up stale remote label announcements.
	// This should match DHTRecordTTL to stay consistent with DHT behavior and prevent
	// our local cache from having stale entries that no longer exist in the DHT.
	RemoteLabelCleanupInterval = 48 * time.Hour

	// ProviderRecordTTL defines the expiration time for CID provider announcements.
	// Provider records (from DHT.Provide()) typically have longer TTL than PutValue records.
	// This is used for cleanup and validation of provider announcements.
	ProviderRecordTTL = 48 * time.Hour

	// RefreshInterval defines how often DHT routing tables are refreshed.
	// This is a shorter interval for maintaining network connectivity.
	RefreshInterval = 30 * time.Second

	// TestRefreshInterval defines a faster refresh interval for testing.
	// This makes tests run faster by reducing wait times for DHT operations.
	TestRefreshInterval = 1 * time.Second
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

	// MinLabelKeyParts defines the minimum number of parts required in a label key after splitting.
	// Format: /type/label/CID splits into ["", "type", "label", "CID"] = 4 parts (empty first due to leading slash).
	MinLabelKeyParts = 4
)
