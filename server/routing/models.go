package routing

// Models for simplicity and documentation when working with Go.
// These are the same Routing API gRPC models.

// This is the actual request over the network.
// Request for querying the DHT in hopes of finding a match.
//
// TODO: expose gRPC interfaces over p2p via streams or RPCs.
type Request struct {
	// Data key, used on the app-layer for data,
	// and on the network-layer for routing.
	// The keys are interpreted by are defined by OASF and Dir.
	// For example, /skills=text,locator=helm-chart.
	// String representation is always sorted.
	//
	// Upper bound limit is set for the number of tags we can query.
	//
	// TODO: find subsets of this on the routing table to point in the direction of serving.
	// Needs to be aggregatable.
	//
	// MULTICAST QUERY
	Key map[string]string

	// Exact peer ID.
	// Forwarded routing layer to get to a specific peer.
	//
	// UNICAST QUERY
	Peer string

	// Content identifier.
	// This is object digest.
	// If we know the CID beforehand, we can request it to speed-up query.
	Digest string

	// Maximum number of hops to perform across nodes.
	// For 0, we will perform exhaustive query, but there is no guarantee that
	// nodes will accept this nor that the request will finish/succeed.
	MaxHops int
}

// Response of the query over a DHT.
type Response struct {
	Peer   string
	Key    map[string]string
	Digest string
}
