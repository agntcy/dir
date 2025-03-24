// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint
package routing

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/agntcy/dir/server/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestPublish_InvalidObject(t *testing.T) {
	r := &routing{}

	t.Run("Invalid object", func(t *testing.T) {
		err := r.Publish(t.Context(), &coretypes.Object{
			Ref:   nil,
			Agent: nil,
		}, true)

		assert.Error(t, err)
		assert.Equal(t, "invalid object reference: <nil>", err.Error())
	})
}

func TestPublishList_ValidSingleSkillQuery(t *testing.T) {
	// Test data
	var (
		testAgent = &coretypes.Agent{
			Skills: []*coretypes.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
			},
		}
		testAgent2 = &coretypes.Agent{
			Skills: []*coretypes.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
				{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
			},
		}

		testRef  = getObjectRef(testAgent)
		testRef2 = getObjectRef(testAgent2)

		validQueriesWithExpectedObjectRef = map[string][]*coretypes.ObjectRef{
			// tests exact lookup for skills
			"/skills/category1/class1": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef.GetDigest(),
				},
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef2.GetDigest(),
				},
			},
			// tests prefix based-lookup for skills
			"/skills/category2": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef2.GetDigest(),
				},
			},
		}
	)

	// create demo network
	mainNode := newTestServer(t.Context(), t, nil)
	r := newTestServer(t.Context(), t, mainNode.server.P2pAddrs())

	// wait for connection
	<-mainNode.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Publish first agent
	err := r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef,
		Agent: testAgent,
	}, true)
	assert.NoError(t, err)

	// Publish second agent
	err = r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef2,
		Agent: testAgent2,
	}, true)
	assert.NoError(t, err)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// list
			refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
				Labels: []string{k},
			})
			assert.NoError(t, err)

			// Collect items from the channel
			var refs []*routingtypes.ListResponse_Item
			for ref := range refsChan {
				refs = append(refs, ref)
			}

			// check if expected refs are present
			assert.Len(t, refs, len(v))

			// check if all expected refs are present
			for _, expectedRef := range v {
				found := false

				for _, ref := range refs {
					if ref.GetRecord().GetDigest() == expectedRef.GetDigest() {
						found = true

						break
					}
				}

				assert.True(t, found, "Expected ref not found: %s", expectedRef.GetDigest())
			}
		})
	}
}

func TestPublishList_ValidMultiSkillQuery(t *testing.T) {
	// Test data
	var (
		testAgent = &coretypes.Agent{
			Skills: []*coretypes.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
				{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
			},
		}

		testRef = getObjectRef(testAgent)
	)

	// create demo network
	mainNode := newTestServer(t.Context(), t, nil)
	r := newTestServer(t.Context(), t, mainNode.server.P2pAddrs())

	// wait for connection
	<-mainNode.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Publish first agent
	err := r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef,
		Agent: testAgent,
	}, true)
	assert.NoError(t, err)

	// Publish second agent
	err = r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef,
		Agent: testAgent,
	}, true)
	assert.NoError(t, err)

	t.Run("Valid multi skill query", func(t *testing.T) {
		// list
		refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
			Labels: []string{"/skills/category1/class1", "/skills/category2/class2"},
		})
		assert.NoError(t, err)

		// Collect items from the channel
		var refs []*routingtypes.ListResponse_Item
		for ref := range refsChan {
			refs = append(refs, ref)
		}

		// check if expected refs are present
		assert.Len(t, refs, 1)

		// check if expected ref is present
		assert.Equal(t, testRef.GetDigest(), refs[0].GetRecord().GetDigest())
	})
}

func getObjectRef(a *coretypes.Agent) *coretypes.ObjectRef {
	raw, _ := json.Marshal(a) //nolint:errchkjson

	return &coretypes.ObjectRef{
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest:      digest.FromBytes(raw).String(),
		Size:        uint64(len(raw)),
		Annotations: a.GetAnnotations(),
	}
}

func toPtr[T any](v T) *T {
	return &v
}

func newTestServer(ctx context.Context, t *testing.T, bootPeers []string) *routing {
	t.Helper()

	r, err := New(ctx, types.NewOptions(
		&config.Config{
			Routing: routingconfig.Config{
				ListenAddress:  routingconfig.DefaultListenddress,
				BootstrapPeers: bootPeers,
			},
		},
		datastore.NewMapDatastore(),
	))
	assert.NoError(t, err)

	routingInstance, ok := r.(*routing)
	assert.True(t, ok, "expected *routing type")

	return routingInstance
}
