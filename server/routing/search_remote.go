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
// Three stages, overlapping rather than sequential: resolve one query label to
// a DHT key and ask who provides it, ask each of those peers which of their
// records match the full query set, then score the answers. Nothing consults a
// local cache of remote announcements — peers answer for themselves, so a peer
// that is down fails at discovery instead of at pull time.
//
// Results are best-effort. The lookup unions the views of whichever custodians
// it reaches inside the budget, and each budget expiring costs recall, not
// correctness.
func (r *routeRemote) searchRemoteRecords(
	ctx context.Context,
	queries []*routingv1.RecordQuery,
	limit uint32,
	minMatchScore uint32,
	outCh chan<- *routingv1.SearchResponse,
) {
	key, label, ok := discoveryKey(queries)
	if !ok {
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

	remoteLogger.Debug("Starting remote search", "label", label, "key", key, "queries", len(queries),
		"minMatchScore", minMatchScore, "limit", limit)

	emitted := make(map[string]struct{})

	for answer := range r.queryProviders(searchCtx, key, request) {
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

	remoteLogger.Debug("Completed remote search", "label", label, "results", len(emitted))
}

// peerAnswer is one peer's reply to the record query.
type peerAnswer struct {
	provider peer.AddrInfo
	matches  []rpc.RecordMatch
}

// queryProviders asks every peer that provides key which of its records match.
//
// The returned channel closes once every discovered peer has answered or the
// context is done.
func (r *routeRemote) queryProviders(ctx context.Context, key cid.Cid, request *rpc.QueryRecordsRequest) <-chan peerAnswer {
	answers := make(chan peerAnswer)
	providers := make(chan peer.AddrInfo, searchProviderBuffer)

	go r.discoverProviders(ctx, key, providers)

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

// discoverProviders drains the DHT provider stream straight into providers.
//
// Nothing slow may happen in this loop. The lookup writes to its channel from
// inside the Kademlia query and the package documents that not reading from it
// blocks the query from progressing, so dialling a peer here would throttle
// discovery itself.
func (r *routeRemote) discoverProviders(ctx context.Context, key cid.Cid, providers chan<- peer.AddrInfo) {
	defer close(providers)

	discoveryCtx, cancel := context.WithTimeout(ctx, SearchDiscoveryTimeout)
	defer cancel()

	self := r.server.Host().ID()
	seen := make(map[peer.ID]struct{})

	// count=0 asks for every provider. Any other value both caps the result and
	// lets the local provider store satisfy the request without touching the
	// network, which would make results depend on what this node has cached.
	for provider := range r.server.DHT().FindProvidersAsync(discoveryCtx, key, 0) {
		// We advertise the labels of the records we hold, so we are a provider
		// of our own results. Search is defined as remote-only; List covers
		// what this node holds.
		if provider.ID == self {
			continue
		}

		// The lookup re-emits a peer whose first sighting carried no addresses,
		// and we would otherwise query it twice.
		if _, ok := seen[provider.ID]; ok {
			continue
		}

		seen[provider.ID] = struct{}{}

		select {
		case providers <- provider:
		case <-ctx.Done():
			return
		}
	}

	remoteLogger.Debug("Provider discovery finished", "key", key, "providers", len(seen))
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

// discoveryKey picks the label to look up and returns its DHT key.
//
// One label is enough. A record satisfying an AND query carries every label the
// query names, and a peer advertises every label of every record it holds, so
// the holder is a provider of each of them; looking up one finds it. Depth is a
// cheap proxy for selectivity — far fewer peers provide /skills/AI/ML than
// /skills/AI, so the deeper key gives a smaller and more accurate candidate set.
func discoveryKey(queries []*routingv1.RecordQuery) (cid.Cid, types.Label, bool) {
	var (
		selected types.Label
		depth    int
	)

	for _, query := range queries {
		label, ok := queryLabel(query)
		if !ok {
			continue
		}

		if labelDepth := strings.Count(label.Value(), "/"); selected == "" || labelDepth > depth {
			selected, depth = label, labelDepth
		}
	}

	if selected == "" {
		return cid.Undef, "", false
	}

	key, err := labelKey(selected)
	if err != nil {
		remoteLogger.Warn("Cannot derive a DHT key from the search label", "label", selected, "error", err)

		return cid.Undef, "", false
	}

	return key, selected, true
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
