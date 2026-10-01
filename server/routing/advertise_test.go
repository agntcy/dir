// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package routing

import (
	"fmt"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// pagedRecordsDB honours limit and offset the way the SQL index does, and
// records the pages it was asked for.
type pagedRecordsDB struct {
	types.DatabaseAPI

	cids   []string
	labels map[string][]types.Label

	pages []int
}

func (p *pagedRecordsDB) GetRecordCIDs(opts ...types.FilterOption) ([]string, error) {
	config := &types.RecordFilters{}
	for _, opt := range opts {
		opt(config)
	}

	p.pages = append(p.pages, config.Offset)

	if config.Offset >= len(p.cids) {
		return nil, nil
	}

	end := min(config.Offset+config.Limit, len(p.cids))

	return p.cids[config.Offset:end], nil
}

func (p *pagedRecordsDB) GetRecordLabels(cids []string) (map[string][]types.Label, error) {
	labels := make(map[string][]types.Label, len(cids))
	for _, cid := range cids {
		labels[cid] = p.labels[cid]
	}

	return labels, nil
}

func TestHeldRecordsPagesThroughTheIndex(t *testing.T) {
	// Two and a bit pages, so the walk has to survive a full page, a partial
	// one, and the boundary between them.
	total := advertisePageSize*2 + 7

	db := &pagedRecordsDB{labels: make(map[string][]types.Label, total)}
	for i := range total {
		cid := fmt.Sprintf("record-%04d", i)
		db.cids = append(db.cids, cid)
		db.labels[cid] = []types.Label{types.Label(fmt.Sprintf("/skills/AI/skill-%04d", i))}
	}

	r := &routeRemote{db: db}

	cids, labels, err := r.publishedRecords(t.Context())
	require.NoError(t, err)

	assert.Equal(t, db.cids, cids)
	assert.Len(t, labels, total)
	assert.Equal(t, []int{0, advertisePageSize, advertisePageSize * 2}, db.pages)
}

func TestHeldRecordsStopsOnAShortPage(t *testing.T) {
	db := &pagedRecordsDB{
		cids:   []string{"record-a", "record-b"},
		labels: map[string][]types.Label{"record-a": {"/skills/AI"}, "record-b": nil},
	}

	r := &routeRemote{db: db}

	cids, labels, err := r.publishedRecords(t.Context())
	require.NoError(t, err)

	assert.Equal(t, db.cids, cids)
	assert.Equal(t, []types.Label{"/skills/AI"}, labels)
	assert.Equal(t, []int{0}, db.pages, "a page shorter than the limit is the last one")
}

func TestHeldRecordsWithoutADatabase(t *testing.T) {
	r := &routeRemote{}

	_, _, err := r.publishedRecords(t.Context())

	require.ErrorIs(t, err, errNoDatabase)
}

// TestHeldButUnpublishedIsNotAdvertised is the privacy guarantee: pushing a
// record makes it servable to whoever knows the CID, and nothing more. Nobody
// learns it exists until it is published.
func TestHeldButUnpublishedIsNotAdvertised(t *testing.T) {
	record := corev1.New(&typesv1alpha1.Record{
		Name:          "private",
		SchemaVersion: "0.7.0",
		Skills:        []*typesv1alpha1.Skill{{Name: "AI/ML"}},
	})

	adapter, err := record.Decode()
	require.NoError(t, err)

	db := newTestDatabase(t)
	node := newTestServer(t, t.Context(), nil, db)

	// Indexing is what a push leaves behind. No Publish call follows.
	require.NoError(t, db.AddRecord(adapter))

	cids, labels, err := node.remote.publishedRecords(t.Context())
	require.NoError(t, err)

	assert.Empty(t, cids, "a held record must not be advertised")
	assert.Empty(t, labels)

	responses, err := node.List(t.Context(), &routingv1.ListRequest{})
	require.NoError(t, err)

	var listed []string
	for response := range responses {
		listed = append(listed, response.GetRecordRef().GetCid())
	}

	assert.Empty(t, listed, "List reports what is published, not what is held")
}

// TestUnpublishSurvivesReadvertising is the point of the published flag. Since
// Kademlia cannot retract a provider record, the only durable effect Unpublish
// can have is to drop the record from every later cycle — including the one a
// restart triggers.
func TestUnpublishSurvivesReadvertising(t *testing.T) {
	kept := corev1.New(&typesv1alpha1.Record{
		Name:          "kept",
		SchemaVersion: "0.7.0",
		Skills:        []*typesv1alpha1.Skill{{Name: "AI/ML"}},
	})
	withdrawn := corev1.New(&typesv1alpha1.Record{
		Name:          "withdrawn",
		SchemaVersion: "0.7.0",
		Skills:        []*typesv1alpha1.Skill{{Name: "AI/NLP"}},
	})

	db := newTestDatabase(t)
	node := newTestServer(t, t.Context(), nil, db)

	publishRecord(t, node, db, kept)
	publishRecord(t, node, db, withdrawn)

	adapter, err := withdrawn.Decode()
	require.NoError(t, err)
	require.NoError(t, node.Unpublish(t.Context(), adapter))

	t.Run("the withdrawn record leaves the published set", func(t *testing.T) {
		cids, labels, err := node.remote.publishedRecords(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{kept.GetCid()}, cids)
		assert.Equal(t, []types.Label{"/skills/AI/ML"}, labels)
	})

	t.Run("its labels stop being announced without a refcount", func(t *testing.T) {
		_, labels, err := node.remote.publishedRecords(t.Context())
		require.NoError(t, err)

		// "/skills/AI" is still there because kept carries it; only the branch
		// unique to the withdrawn record disappears.
		expanded := expandLabels(labels)
		assert.Contains(t, expanded, types.Label("/skills/AI"))
		assert.NotContains(t, expanded, types.Label("/skills/AI/NLP"))
	})

	t.Run("List stops reporting it", func(t *testing.T) {
		responses, err := node.List(t.Context(), &routingv1.ListRequest{})
		require.NoError(t, err)

		var cids []string
		for response := range responses {
			cids = append(cids, response.GetRecordRef().GetCid())
		}

		assert.Equal(t, []string{kept.GetCid()}, cids)
	})

	t.Run("publishing again restores it", func(t *testing.T) {
		require.NoError(t, node.Publish(t.Context(), adapter))

		cids, _, err := node.remote.publishedRecords(t.Context())
		require.NoError(t, err)

		assert.ElementsMatch(t, []string{kept.GetCid(), withdrawn.GetCid()}, cids)
	})
}

// TestAdvertiseOnStartup is the restart case: a node that published a record in
// an earlier lifetime has to re-announce it on boot, because provider records
// expire and the reprovide ticker does not fire when it is created.
func TestAdvertiseOnStartup(t *testing.T) {
	record := corev1.New(&typesv1alpha1.Record{
		Name:          "startup-agent",
		SchemaVersion: "0.7.0",
		Skills:        []*typesv1alpha1.Skill{{Name: "AI/ML"}},
	})

	adapter, err := record.Decode()
	require.NoError(t, err)

	// State a previous run would have left behind: indexed and published. The
	// node starts after this, so only the startup pass can advertise it.
	db := newTestDatabase(t)
	require.NoError(t, db.AddRecord(adapter))
	require.NoError(t, db.SetRecordPublished(record.GetCid(), true))

	holder := newTestServer(t, t.Context(), nil, db)
	searcher := newTestServer(t, t.Context(), holder.remote.server.P2pAddrs(), nil)

	<-holder.remote.server.DHT().RefreshRoutingTable()
	<-searcher.remote.server.DHT().RefreshRoutingTable()

	// The holder waits for its first peer before advertising, so how long this
	// takes depends on when the searcher shows up in its routing table.
	require.Eventually(t, func() bool {
		responses := collectSearch(t, searcher, &routingv1.SearchRequest{
			Queries: []*routingv1.RecordQuery{skillQuery("AI")},
		})

		return len(responses) == 1 && responses[0].GetRecordRef().GetCid() == record.GetCid()
	}, 60*time.Second, 2*time.Second, "startup advertisement never reached the searcher")
}
