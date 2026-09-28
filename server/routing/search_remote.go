// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"strings"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/routing/internal/p2p"
	"github.com/agntcy/dir/server/routing/rpc"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
)

// searchRemoteRecords finds records held by other peers and streams the matches.
//
// Three stages, overlapping rather than sequential: resolve the query labels to
// DHT keys and ask who provides them, ask each of those peers which of their
// records match the full query set, then score the answers. Nothing consults a
// local cache of remote announcements — peers answer for themselves, so a peer
// that is down fails at discovery instead of at pull time.
//
// Results are best-effort. The lookups union the views of whichever custodians
// they reach inside the budget, and each budget expiring costs recall, not
// correctness.
func (r *routeRemote) searchRemoteRecords(
	ctx context.Context,
	queries []*routingv1.RecordQuery,
	limit uint32,
	minMatchScore uint32,
	outCh chan<- *routingv1.SearchResponse,
) {
	targets := discoveryKeys(queries)
	if len(targets) == 0 {
		remoteLogger.Warn("Remote search needs a skill, domain, module or locator query to look up",
			"queries", len(queries))

		return
	}

	searchCtx, cancel := context.WithTimeout(ctx, SearchTimeout)
	defer cancel()

	request := &rpc.QueryRecordsRequest{
		Queries: peerQueries(queries),
		Limit:   peerLimit(limit, minMatchScore),
	}

	remoteLogger.Debug("Starting remote search", "labels", targetLabels(targets), "queries", len(queries),
		"minMatchScore", minMatchScore, "limit", limit)

	emitted := make(map[string]struct{})

	for answer := range r.queryProviders(searchCtx, targets, request) {
		for _, match := range answer.matches {
			if _, done := emitted[match.Cid]; done {
				continue
			}

			matched, score := scoreMatch(queries, toLabels(match.Labels))
			if score < minMatchScore {
				remoteLogger.Debug("Discarding record below the match threshold",
					"cid", match.Cid, "peer", answer.provider.ID, "score", score)

				continue
			}

			select {
			case outCh <- &routingv1.SearchResponse{
				RecordRef:    &corev1.RecordRef{Cid: match.Cid},
				Peer:         r.peerInfo(answer.provider),
				MatchQueries: matched,
				MatchScore:   score,
			}:
			case <-searchCtx.Done():
				return
			}

			emitted[match.Cid] = struct{}{}

			// Returning cancels searchCtx, which unwinds the lookup and the
			// workers still waiting to report.
			if limit > 0 && safeIntToUint32(len(emitted)) >= limit {
				remoteLogger.Debug("Remote search reached the requested limit", "limit", limit)

				return
			}
		}
	}

	remoteLogger.Debug("Completed remote search", "labels", targetLabels(targets), "results", len(emitted))
}

// peerAnswer is one peer's reply to the record query.
type peerAnswer struct {
	provider peer.AddrInfo
	matches  []rpc.RecordMatch
}

// queryProviders asks every peer providing any of the keys which of its records
// match.
//
// The returned channel closes once every discovered peer has answered or the
// context is done.
func (r *routeRemote) queryProviders(ctx context.Context, targets []discoveryTarget, request *rpc.QueryRecordsRequest) <-chan peerAnswer {
	answers := make(chan peerAnswer)
	providers := make(chan peer.AddrInfo, searchProviderBuffer)

	go r.discoverProviders(ctx, targets, providers)

	var wg sync.WaitGroup

	for range searchPeerWorkers {
		wg.Go(func() {
			for provider := range providers {
				matches := r.queryPeer(ctx, provider, request)
				if len(matches) == 0 {
					continue
				}

				select {
				case answers <- peerAnswer{provider: provider, matches: matches}:
				case <-ctx.Done():
					return
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(answers)
	}()

	return answers
}

// discoverProviders looks every key up and unions the peers into providers.
//
// The lookups share one budget and run concurrently, so covering several labels
// costs the same wall clock as covering one rather than multiplying it. A single
// collector owns the dedupe: the same peer routinely provides more than one of
// the keys, and dialling it once per key would waste the query pool.
func (r *routeRemote) discoverProviders(ctx context.Context, targets []discoveryTarget, providers chan<- peer.AddrInfo) {
	defer close(providers)

	discoveryCtx, cancel := context.WithTimeout(ctx, SearchDiscoveryTimeout)
	defer cancel()

	pending := make(chan discoveryTarget, len(targets))
	for _, target := range targets {
		pending <- target
	}

	close(pending)

	found := make(chan peer.AddrInfo)

	var wg sync.WaitGroup

	for range min(len(targets), searchDiscoveryWorkers) {
		wg.Go(func() {
			for target := range pending {
				r.findProviders(discoveryCtx, target, found)
			}
		})
	}

	go func() {
		wg.Wait()
		close(found)
	}()

	self := r.server.Host().ID()
	seen := make(map[peer.ID]struct{})

	for provider := range found {
		// We advertise the labels of the records we hold, so we are a provider
		// of our own results. Search is defined as remote-only; List covers
		// what this node holds.
		if provider.ID == self {
			continue
		}

		// A lookup re-emits a peer whose first sighting carried no addresses,
		// and two lookups routinely land on the same peer, so without this we
		// would query it more than once.
		if _, ok := seen[provider.ID]; ok {
			continue
		}

		seen[provider.ID] = struct{}{}

		select {
		case providers <- provider:
		case <-ctx.Done():
			// The deferred cancel unblocks the lookups still waiting to report,
			// which lets them finish and the collector goroutine close found.
			return
		}
	}

	remoteLogger.Debug("Provider discovery finished", "keys", len(targets), "providers", len(seen))
}

// findProviders drains one key's provider stream into found.
//
// Nothing slow may happen in this loop. The lookup writes to its channel from
// inside the Kademlia query and the package documents that not reading from it
// blocks the query from progressing, so dialling a peer here would throttle
// discovery itself.
func (r *routeRemote) findProviders(ctx context.Context, target discoveryTarget, found chan<- peer.AddrInfo) {
	// count=0 asks for every provider. Any other value both caps the result and
	// lets the local provider store satisfy the request without touching the
	// network, which would make results depend on what this node has cached.
	for provider := range r.server.DHT().FindProvidersAsync(ctx, target.key, 0) {
		select {
		case found <- provider:
		case <-ctx.Done():
			return
		}
	}
}

// queryPeer asks one peer which of its records match, returning nothing if it
// cannot answer. A provider record outlives the peer that wrote it, so an
// unreachable peer is expected rather than exceptional.
func (r *routeRemote) queryPeer(ctx context.Context, provider peer.AddrInfo, request *rpc.QueryRecordsRequest) []rpc.RecordMatch {
	peerCtx, cancel := context.WithTimeout(ctx, SearchPeerTimeout)
	defer cancel()

	matches, err := r.service.QueryRecords(peerCtx, provider.ID, request)
	if err != nil {
		remoteLogger.Debug("Provider did not answer the record query", "peer", provider.ID, "error", err)

		return nil
	}

	return matches
}

// discoveryTarget is a label to look up and the DHT key it resolves to.
type discoveryTarget struct {
	key   cid.Cid
	label types.Label
}

// discoveryKeys picks the labels to look up and resolves them to DHT keys.
//
// Every query needs covering, not just one. Search is OR-with-threshold, so a
// record qualifies on a single query and the peers holding matches for
// different queries may be entirely disjoint. Resolving one label would leave
// the rest undiscoverable, and silently: a peer nobody asks reports nothing.
//
// Descendants are dropped as redundant. A holder advertises every ancestor of
// every label it holds, so the providers of "/skills/AI" already include every
// provider of "/skills/AI/ML" and resolving both finds nothing extra. What
// survives is the shallowest label of each ancestor chain, usually far fewer
// keys than there are queries. Namespaces never share ancestry, so skills,
// domains, modules and locators each contribute at least one.
//
// Reaching a peer through a broader ancestor costs only the round trip: the
// peer applies the real queries against its own records, so one that matched
// nothing but the ancestor answers with nothing.
func discoveryKeys(queries []*routingv1.RecordQuery) []discoveryTarget {
	labels := distinctQueryLabels(queries)

	searched := make(map[types.Label]struct{}, len(labels))
	for _, label := range labels {
		searched[label] = struct{}{}
	}

	targets := make([]discoveryTarget, 0, len(labels))

	for _, label := range labels {
		if hasAncestorIn(label, searched) {
			continue
		}

		key, err := labelKey(label)
		if err != nil {
			remoteLogger.Warn("Cannot derive a DHT key from the search label", "label", label, "error", err)

			continue
		}

		targets = append(targets, discoveryTarget{key: key, label: label})
	}

	return targets
}

// distinctQueryLabels collects the labels the queries name in first-seen order,
// normalized so that "/skills/AI" and "/skills/AI/" count as one.
//
// Order is kept rather than using a bare set, so the keys — and therefore the
// order results arrive in — do not vary run to run.
func distinctQueryLabels(queries []*routingv1.RecordQuery) []types.Label {
	seen := make(map[types.Label]struct{}, len(queries))
	labels := make([]types.Label, 0, len(queries))

	for _, query := range queries {
		label, ok := queryLabel(query)
		if !ok {
			continue
		}

		normalized := types.Label(normalizeLabel(label))
		if normalized == "" {
			continue
		}

		if _, ok := seen[normalized]; ok {
			continue
		}

		seen[normalized] = struct{}{}

		labels = append(labels, normalized)
	}

	return labels
}

// hasAncestorIn reports whether a broader label is being searched for too,
// which makes this one redundant as a lookup key.
func hasAncestorIn(label types.Label, searched map[types.Label]struct{}) bool {
	// expandLabel yields the label itself first and its ancestors after, and
	// nothing at all for a bare namespace — which labelKey rejects anyway.
	expanded := expandLabel(label)
	if len(expanded) == 0 {
		return false
	}

	for _, ancestor := range expanded[1:] {
		if _, ok := searched[ancestor]; ok {
			return true
		}
	}

	return false
}

// targetLabels lists the labels being resolved, for logging.
func targetLabels(targets []discoveryTarget) []string {
	labels := make([]string, len(targets))
	for i, target := range targets {
		labels[i] = target.label.String()
	}

	return labels
}

// peerQueries converts the request into its wire form, dropping queries that
// name no label. An unspecified query matches everything, so it contributes to
// the score without narrowing what a peer should return.
func peerQueries(queries []*routingv1.RecordQuery) []rpc.RecordQuery {
	converted := make([]rpc.RecordQuery, 0, len(queries))

	for _, query := range queries {
		label, ok := queryLabel(query)
		if !ok {
			continue
		}

		converted = append(converted, rpc.RecordQuery{
			Type:  label.Type().String(),
			Value: label.Value(),
		})
	}

	return converted
}

// peerLimit decides how many records to ask each peer for.
//
// Normally the caller's limit: every record a peer returns matched at least one
// query, which already clears the default threshold, so none of them are wasted.
// A higher threshold is scored here and not there, so the peer has to offer more
// candidates than the caller will keep; zero lets it apply its own cap.
func peerLimit(limit uint32, minMatchScore uint32) uint32 {
	if minMatchScore > DefaultMinMatchScore {
		return 0
	}

	return limit
}

// queryLabel maps a query onto the label it searches for.
func queryLabel(query *routingv1.RecordQuery) (types.Label, bool) {
	value := strings.TrimSpace(query.GetValue())
	if value == "" {
		return "", false
	}

	labelType, ok := queryLabelType(query.GetType())
	if !ok {
		return "", false
	}

	return labelType.LabelKey(value), true
}

func queryLabelType(queryType routingv1.RecordQueryType) (types.LabelType, bool) {
	switch queryType {
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL:
		return types.LabelTypeSkill, true
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN:
		return types.LabelTypeDomain, true
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_MODULE:
		return types.LabelTypeModule, true
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_LOCATOR:
		return types.LabelTypeLocator, true
	case routingv1.RecordQueryType_RECORD_QUERY_TYPE_UNSPECIFIED:
		return types.LabelTypeUnknown, false
	default:
		return types.LabelTypeUnknown, false
	}
}

// scoreMatch counts how many queries the record's labels satisfy. Queries are
// OR'd: the count is the score the caller thresholds on.
func scoreMatch(queries []*routingv1.RecordQuery, labels []types.Label) ([]*routingv1.RecordQuery, uint32) {
	if len(queries) == 0 || len(labels) == 0 {
		return nil, 0
	}

	matched := make([]*routingv1.RecordQuery, 0, len(queries))

	for _, query := range queries {
		if QueryMatchesLabels(query, labels) {
			matched = append(matched, query)
		}
	}

	return matched, safeIntToUint32(len(matched))
}

func toLabels(values []string) []types.Label {
	labels := make([]types.Label, len(values))
	for i, value := range values {
		labels[i] = types.Label(value)
	}

	return labels
}

// peerInfo describes where a provider can be reached, advertising its Directory
// API (/dir/) and OCI registry (/oci/) endpoints in prefixed multiaddr form so
// the consumer can tell them apart. Either may be missing.
//
// Addresses come from the provider record, falling back to the peerstore for a
// record that arrived without any — a peer we are already connected to has told
// us its addresses over identify.
func (r *routeRemote) peerInfo(provider peer.AddrInfo) *routingv1.Peer {
	known := provider.Addrs
	if len(known) == 0 {
		known = r.server.Host().Peerstore().Addrs(provider.ID)
	}

	addrs := make([]string, 0, 2) //nolint:mnd // dir + oci

	for _, protocol := range []struct {
		name string
		code int
	}{
		{p2p.DirProtocol, p2p.DirProtocolCode},
		{p2p.OciProtocol, p2p.OciProtocolCode},
	} {
		if value := extractProtocolValue(known, protocol.code); value != "" {
			addrs = append(addrs, "/"+protocol.name+"/"+value)
		}
	}

	return &routingv1.Peer{
		Id:    provider.ID.String(),
		Addrs: addrs,
	}
}
