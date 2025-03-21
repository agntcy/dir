// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint
package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/internal/p2p"
	"github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestPublish_InvalidRef(t *testing.T) {
	r := &routing{ds: nil}
	invalidRef := &coretypes.ObjectRef{Type: "invalid"}

	t.Run("Invalid ref: "+invalidRef.GetType(), func(t *testing.T) {
		err := r.Publish(t.Context(), invalidRef, nil)
		assert.Error(t, err)
		assert.Equal(t, "invalid object type: "+invalidRef.GetType(), err.Error())
	})
}

func TestList_InvalidQuery(t *testing.T) {
	r := &routing{}
	invalidQueries := []string{
		"",
		"/",
		"/agents",
		"/agents/agentX",
		"/skills/",
		"/locators",
		"skills/",
		"locators/",
	}

	for _, q := range invalidQueries {
		t.Run("Invalid query: "+q, func(t *testing.T) {
			_, err := r.List(t.Context(), q)
			assert.Error(t, err)
			assert.Equal(t, "invalid query: "+q, err.Error())
		})
	}
}

func TestPublishList_ValidQuery(t *testing.T) {
	// Test data
	var (
		testAgent = &coretypes.Agent{
			Skills: []*coretypes.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
			},
			Locators: []*coretypes.Locator{
				{Type: "type1", Url: "url1"},
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
			"/skills/category2/class2": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef2.GetDigest(),
				},
			},
			"/locators/type1": {
				{
					Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
					Digest: testRef.GetDigest(),
				},
			},
		}
	)

	// create demo network
	mainNode, err := p2p.NewMockServer(t.Context(), protocol.ID(ProtocolPrefix))
	assert.NoError(t, err)

	// create in-memory datastore
	ds := datastore.NewMapDatastore()
	r := &routing{
		ds:     ds,
		server: mainNode,
	}
	assert.NoError(t, err)
	<-mainNode.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Publish first agent
	err = r.Publish(t.Context(), testRef, testAgent)
	assert.NoError(t, err)

	// Publish second agent
	err = r.Publish(t.Context(), testRef2, testAgent2)
	assert.NoError(t, err)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// list
			refs, err := r.List(t.Context(), k)
			assert.NoError(t, err)

			// check if expected refs are present
			assert.Len(t, refs, len(v))

			for _, ref := range refs {
				for _, r := range v {
					if ref.GetDigest() == r.GetDigest() {
						break
					}
				}
			}
		})
	}
}

func TestRouting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create demo network
	server, err := p2p.NewMockServer(ctx, protocol.ID(ProtocolPrefix))
	assert.NoError(t, err)

	// wait for server ready state
	time.Sleep(1 * time.Second)

	// announce
	err = server.DHT().Provide(ctx, keyToCid("/skills/helm-chart/my-agent"), true)
	assert.NoError(t, err)

	// discover
	ch := server.DHT().FindProvidersAsync(ctx, keyToCid("/skills/helm-chart/my-agent"), 10)
	assert.NoError(t, err)
	found := false
	for data := range ch {
		fmt.Println(data.String())
		found = true
	}
	assert.True(t, found)

	// get providers
	peers, err := server.DHT().GetClosestPeers(ctx, "/asf")
	assert.NoError(t, err)
	assert.True(t, len(peers) > 0)
}

func getObjectRef(a *coretypes.Agent) *coretypes.ObjectRef {
	raw, _ := json.Marshal(a) //nolint:errchkjson

	return &coretypes.ObjectRef{
		Type:        coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
		Digest:      digest.FromBytes(raw).String(),
		Size:        uint64(len(raw)),
		Annotations: a.Annotations,
	}
}

func toPtr[T any](v T) *T {
	return &v
}
