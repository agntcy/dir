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
	// RepublishInterval defines how often published records are re-advertised
	// so their provider records, for both CIDs and labels, stay alive.
	RepublishInterval = 36 * time.Hour
	// advertisePollInterval is how often a node with an empty routing table
	// rechecks for a peer to advertise to.
	advertisePollInterval = 5 * time.Second
	// RefreshInterval defines how often DHT routing tables are refreshed.
	// This is a shorter interval for maintaining network connectivity.
	RefreshInterval = 30 * time.Second
	// ProviderCountTimeout bounds a single provider-count lookup. The caller
	// iterates every local record, so this is a latency budget per CID rather
	// than a correctness knob: cutting a lookup short undercounts providers.
	ProviderCountTimeout = 10 * time.Second
)

// Budgets for a remote search. They overlap rather than run in sequence: the
// provider lookup and the peer queries it feeds are concurrent, and every one
// of them is capped by the overall search budget.
//
// Results stream, so these bound latency rather than correctness — an expiring
// budget costs recall.
const (
	// SearchTimeout bounds a whole remote search, discovery and peer queries.
	SearchTimeout = 30 * time.Second

	// SearchDiscoveryTimeout bounds the DHT provider lookup. It runs to
	// completion rather than stopping at a target count, so it needs a deadline
	// of its own; peers already found keep being queried after it expires.
	SearchDiscoveryTimeout = 15 * time.Second

	// SearchPeerTimeout bounds one peer's query. Provider records outlive the
	// peers that wrote them, so some of these will always time out.
	SearchPeerTimeout = 10 * time.Second
)

// Protocol constants for libp2p DHT and discovery.
const (
	// ProtocolPrefix is the prefix used for DHT protocol identification.
	// The DHT appends "/kad/1.0.0", so peers speak "/dir/2/kad/1.0.0".
	// v2 nodes must not share a routing table with v1: they advertise label
	// provider records that a v1 node would misread as content CIDs.
	ProtocolPrefix = "/dir/2"

	// ProtocolRendezvous is the rendezvous string used for peer discovery.
	ProtocolRendezvous = "dir/2/connect"
)

// Validation rules and limits.
const (
	// MaxHops defines the maximum number of hops allowed in distributed queries.
	MaxHops = 20

	// advertiseConcurrency bounds how many DHT Provide calls run at once.
	// Each one is a full Kademlia lookup followed by K AddProvider sends, so a
	// record carrying several deep skills would otherwise serialise into tens
	// of seconds on a synchronous publish.
	advertiseConcurrency = 8

	// searchPeerWorkers bounds how many providers are queried at once.
	searchPeerWorkers = 8

	// searchProviderBuffer decouples the provider lookup from those workers.
	// The lookup stalls while nobody reads its channel, so discovery has to be
	// able to run ahead of the queries it feeds.
	searchProviderBuffer = 64

	// advertisePageSize bounds how many published records are loaded at a time when
	// enumerating what to advertise. GetRecords applies no LIMIT when the limit
	// is zero, so an unpaged call would read the whole corpus into memory.
	advertisePageSize = 500

	// maxListCandidates caps how many records one query of a multi-query list
	// pulls before the results are intersected. The requested limit cannot be
	// pushed into those queries without dropping valid matches, so this bounds
	// the work instead.
	maxListCandidates = 10000

	// DefaultMinMatchScore defines the minimum allowed match score for production safety.
	// Per proto specification: "If not set, it will return records that match at least one query".
	// Any value below this threshold is automatically corrected to this value.
	DefaultMinMatchScore = 1
)

const ResultChannelBufferSize = 100
