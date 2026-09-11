// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"context"
	"slices"
	"testing"
	"time"

	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// heldRecordsDB is what a node answers peer queries with. Which records match
// which filters is settled in the RPC package's own tests; here the point is
// that the answer reaches the searcher and gets scored.
type heldRecordsDB struct {
	types.DatabaseAPI

	labels map[string][]types.Label
}

func (h *heldRecordsDB) GetRecordCIDs(...types.FilterOption) ([]string, error) {
	cids := make([]string, 0, len(h.labels))
	for cid := range h.labels {
		cids = append(cids, cid)
	}

	slices.Sort(cids)

	return cids, nil
}

func (h *heldRecordsDB) GetRecordLabels(cids []string) (map[string][]types.Label, error) {
	labels := make(map[string][]types.Label, len(cids))

	for _, cid := range cids {
		if recordLabels, ok := h.labels[cid]; ok {
			labels[cid] = recordLabels
		}
	}

	return labels, nil
}

func (h *heldRecordsDB) advertised() []types.Label {
	var labels []types.Label
	for _, recordLabels := range h.labels {
		labels = append(labels, recordLabels...)
	}

	return expandLabels(labels)
}

// newSearchNetwork starts a node holding the given records and a second node
// bootstrapped off it, advertises the holder's labels, and returns both — the
// holder first.
func newSearchNetwork(t *testing.T, held *heldRecordsDB) (*route, *route) {
	t.Helper()

	ctx := t.Context()

	holder := newTestServer(t, ctx, nil, held)
	searcher := newTestServer(t, ctx, holder.remote.server.P2pAddrs(), nil)

	<-holder.remote.server.DHT().RefreshRoutingTable()
	<-searcher.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	provideCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	require.Zero(t, holder.remote.provideLabels(provideCtx, held.advertised()),
		"every label should reach the DHT in a two-node network")

	return holder, searcher
}

func collectSearch(t *testing.T, node *route, req *routingv1.SearchRequest) []*routingv1.SearchResponse {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), SearchTimeout+10*time.Second)
	defer cancel()

	responses, err := node.Search(ctx, req)
	require.NoError(t, err)

	var collected []*routingv1.SearchResponse
	for response := range responses {
		collected = append(collected, response)
	}

	return collected
}

func responseCIDs(responses []*routingv1.SearchResponse) []string {
	cids := make([]string, 0, len(responses))
	for _, response := range responses {
		cids = append(cids, response.GetRecordRef().GetCid())
	}

	slices.Sort(cids)

	return cids
}

func TestRemoteSearchOverTheNetwork(t *testing.T) {
	held := &heldRecordsDB{labels: map[string][]types.Label{
		"record-ml":  {"/skills/AI/ML", "/domains/healthcare"},
		"record-nlp": {"/skills/AI/NLP"},
	}}

	holder, searcher := newSearchNetwork(t, held)

	t.Run("a parent skill reaches records tagged with its descendants", func(t *testing.T) {
		responses := collectSearch(t, searcher, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{skillQuery("AI")},
		})

		assert.Equal(t, []string{"record-ml", "record-nlp"}, responseCIDs(responses))

		for _, response := range responses {
			assert.Equal(t, holder.remote.server.Host().ID().String(), response.GetPeer().GetId())
			assert.Equal(t, uint32(1), response.GetMatchScore())
		}
	})

	t.Run("a record matching more queries scores higher", func(t *testing.T) {
		responses := collectSearch(t, searcher, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{
				skillQuery("AI"),
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, Value: "healthcare"},
			},
		})

		scores := make(map[string]uint32, len(responses))
		for _, response := range responses {
			scores[response.GetRecordRef().GetCid()] = response.GetMatchScore()
		}

		assert.Equal(t, map[string]uint32{"record-ml": 2, "record-nlp": 1}, scores)
	})

	t.Run("the threshold drops records that match too few queries", func(t *testing.T) {
		responses := collectSearch(t, searcher, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{
				skillQuery("AI"),
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_DOMAIN, Value: "healthcare"},
			},
			MinMatchScore: new(uint32(2)),
		})

		assert.Equal(t, []string{"record-ml"}, responseCIDs(responses))
	})

	t.Run("the limit caps the results", func(t *testing.T) {
		responses := collectSearch(t, searcher, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{skillQuery("AI")},
			Limit:   new(uint32(1)),
		})

		assert.Len(t, responses, 1)
	})

	t.Run("an unadvertised label finds nothing", func(t *testing.T) {
		responses := collectSearch(t, searcher, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{skillQuery("Robotics")},
		})

		assert.Empty(t, responses)
	})

	t.Run("a node does not return its own records", func(t *testing.T) {
		// The holder provides every one of these labels, so without the
		// self-check it would query itself and return everything it holds.
		responses := collectSearch(t, holder, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{skillQuery("AI")},
		})

		assert.Empty(t, responses)
	})
}

func TestRemoteSearchWhenTheProviderCannotAnswer(t *testing.T) {
	// A node with no database rejects the query. The searcher has to finish
	// cleanly rather than hang on a provider that discovery did find.
	ctx := t.Context()

	holder := newTestServer(t, ctx, nil, nil)
	searcher := newTestServer(t, ctx, holder.remote.server.P2pAddrs(), nil)

	<-holder.remote.server.DHT().RefreshRoutingTable()
	<-searcher.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	provideCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	require.Zero(t, holder.remote.provideLabels(provideCtx, expandLabels([]types.Label{"/skills/AI/ML"})))

	responses := collectSearch(t, searcher, &routingv1.SearchRequest{
		Queries: []*routingv1.RecordQuery{skillQuery("AI")},
	})

	assert.Empty(t, responses)
}
