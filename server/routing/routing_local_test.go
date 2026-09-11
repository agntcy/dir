// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint,wsl
package routing

import (
	"strings"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// publishRecord indexes a record and announces it, which is what a completed
// push followed by a publish leaves behind.
func publishRecord(t *testing.T, r *route, db types.DatabaseAPI, record *corev1.Record) {
	t.Helper()

	adapter, err := record.Decode()
	require.NoError(t, err)

	require.NoError(t, db.AddRecord(adapter))
	require.NoError(t, r.Publish(t.Context(), adapter))
}

func TestPublish_NilRecord(t *testing.T) {
	r := &route{}

	err := r.Publish(t.Context(), nil)

	assert.Error(t, err)
	assert.ErrorContains(t, err, "record is required")
}

func TestPublishList_ValidSingleSkillQuery(t *testing.T) {
	var (
		testRecord = corev1.New(&typesv1alpha1.Record{
			Name:          "test-agent-1",
			SchemaVersion: "0.7.0",
			Skills: []*typesv1alpha1.Skill{
				{Name: "category1/class1"},
			},
		})
		testRecord2 = corev1.New(&typesv1alpha1.Record{
			Name:          "test-agent-2",
			SchemaVersion: "0.7.0",
			Skills: []*typesv1alpha1.Skill{
				{Name: "category1/class1"},
				{Name: "category2/class2"},
			},
		})

		testRef  = &corev1.RecordRef{Cid: testRecord.GetCid()}
		testRef2 = &corev1.RecordRef{Cid: testRecord2.GetCid()}

		validQueriesWithExpectedObjectRef = map[string][]*corev1.RecordRef{
			// tests exact lookup for skills
			"/skills/category1/class1": {
				{Cid: testRef.GetCid()},
				{Cid: testRef2.GetCid()},
			},
			// tests prefix based-lookup for skills
			"/skills/category2": {
				{Cid: testRef2.GetCid()},
			},
		}
	)

	// create demo network
	db := newTestDatabase(t)
	mainNode := newTestServer(t, t.Context(), nil, nil)
	r := newTestServer(t, t.Context(), mainNode.remote.server.P2pAddrs(), db)

	// wait for connection
	<-mainNode.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	publishRecord(t, r, db, testRecord)
	publishRecord(t, r, db, testRecord2)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// Convert label to RecordQuery
			var queries []*routingv1.RecordQuery

			if after, ok := strings.CutPrefix(k, "/skills/"); ok {
				skillName := after
				queries = append(queries, &routingv1.RecordQuery{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: skillName,
				})
			}

			// list
			refsChan, err := r.List(t.Context(), &routingv1.ListRequest{
				Queries: queries,
			})
			assert.NoError(t, err)

			// Collect items from the channel
			var refs []*routingv1.ListResponse
			for ref := range refsChan {
				refs = append(refs, ref)
			}

			// check if expected refs are present
			assert.Len(t, refs, len(v))

			// check if all expected refs are present
			for _, expectedRef := range v {
				found := false

				for _, ref := range refs {
					if ref.GetRecordRef().GetCid() == expectedRef.GetCid() {
						found = true

						break
					}
				}

				assert.True(t, found, "Expected ref not found: %s", expectedRef.GetCid())
			}
		})
	}

	// Unpublish second record
	adapterUnpub, err := testRecord2.Decode()
	assert.NoError(t, err)
	err = r.Unpublish(t.Context(), adapterUnpub)
	assert.NoError(t, err)
	assert.NoError(t, db.RemoveRecord(testRecord2.GetCid()))

	// Try to list second record using RecordQuery
	refsChan, err := r.List(t.Context(), &routingv1.ListRequest{
		Queries: []*routingv1.RecordQuery{
			{
				Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
				Value: "category2",
			},
		},
	})
	assert.NoError(t, err)

	// Collect items from the channel
	var refs []*routingv1.ListResponse
	for ref := range refsChan {
		refs = append(refs, ref)
	}

	// check no refs are present
	assert.Len(t, refs, 0)
}

func TestPublishList_ValidMultiSkillQuery(t *testing.T) {
	// Test data
	var (
		testRecord = corev1.New(&typesv1alpha1.Record{
			Name:          "test-agent-multi",
			SchemaVersion: "0.7.0",
			Skills: []*typesv1alpha1.Skill{
				{Name: "category1/class1"},
				{Name: "category2/class2"},
			},
		})
		testRef = &corev1.RecordRef{Cid: testRecord.GetCid()}
	)

	// create demo network
	db := newTestDatabase(t)
	mainNode := newTestServer(t, t.Context(), nil, nil)
	r := newTestServer(t, t.Context(), mainNode.remote.server.P2pAddrs(), db)

	// wait for connection
	<-mainNode.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	publishRecord(t, r, db, testRecord)

	t.Run("Valid multi skill query", func(t *testing.T) {
		// list with multiple RecordQueries (AND logic)
		refsChan, err := r.List(t.Context(), &routingv1.ListRequest{
			Queries: []*routingv1.RecordQuery{
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "category1/class1",
				},
				{
					Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
					Value: "category2/class2",
				},
			},
		})
		assert.NoError(t, err)

		// Collect items from the channel
		var refs []*routingv1.ListResponse
		for ref := range refsChan {
			refs = append(refs, ref)
		}

		// check if expected refs are present
		assert.Len(t, refs, 1)

		// check if expected ref is present
		assert.Equal(t, testRef.GetCid(), refs[0].GetRecordRef().GetCid())
	})
}

// TestLocalList covers the parts of List that are not about a single skill
// filter: unfiltered listing, the limit, the returned label set, and the AND
// across queries.
func TestLocalList(t *testing.T) {
	bothSkills := corev1.New(&typesv1alpha1.Record{
		Name:          "both-skills",
		SchemaVersion: "0.7.0",
		Skills: []*typesv1alpha1.Skill{
			{Name: "category1/class1"},
			{Name: "category2/class2"},
		},
	})
	oneSkill := corev1.New(&typesv1alpha1.Record{
		Name:          "one-skill",
		SchemaVersion: "0.7.0",
		Skills:        []*typesv1alpha1.Skill{{Name: "category1/class1"}},
	})

	db := newTestDatabase(t)
	node := newTestServer(t, t.Context(), nil, db)

	publishRecord(t, node, db, bothSkills)
	publishRecord(t, node, db, oneSkill)

	list := func(t *testing.T, req *routingv1.ListRequest) []*routingv1.ListResponse {
		t.Helper()

		responses, err := node.List(t.Context(), req)
		require.NoError(t, err)

		var collected []*routingv1.ListResponse
		for response := range responses {
			collected = append(collected, response)
		}

		return collected
	}

	t.Run("no queries returns everything held", func(t *testing.T) {
		assert.Len(t, list(t, &routingv1.ListRequest{}), 2)
	})

	t.Run("limit caps the results", func(t *testing.T) {
		assert.Len(t, list(t, &routingv1.ListRequest{Limit: new(uint32(1))}), 1)
	})

	t.Run("queries AND rather than OR", func(t *testing.T) {
		// oneSkill satisfies the first query only, so a union would wrongly
		// return it alongside bothSkills.
		responses := list(t, &routingv1.ListRequest{
			Queries: []*routingv1.RecordQuery{
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "category1/class1"},
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "category2/class2"},
			},
		})

		require.Len(t, responses, 1)
		assert.Equal(t, bothSkills.GetCid(), responses[0].GetRecordRef().GetCid())
	})

	t.Run("responses carry the full label set", func(t *testing.T) {
		responses := list(t, &routingv1.ListRequest{
			Queries: []*routingv1.RecordQuery{
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "category2"},
			},
		})

		require.Len(t, responses, 1)
		assert.ElementsMatch(t,
			[]string{"/skills/category1/class1", "/skills/category2/class2"},
			responses[0].GetLabels())
	})

	t.Run("unmatched query returns nothing", func(t *testing.T) {
		assert.Empty(t, list(t, &routingv1.ListRequest{
			Queries: []*routingv1.RecordQuery{
				{Type: routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL, Value: "category3"},
			},
		}))
	})
}
