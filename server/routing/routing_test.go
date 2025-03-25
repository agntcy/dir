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
	"github.com/agntcy/dir/server/config"
	routingconfig "github.com/agntcy/dir/server/routing/config"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-datastore"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

// func TestPublish_InvalidRef(t *testing.T) {
// 	r := &routing{}
// 	invalidRef := &coretypes.ObjectRef{Type: "invalid"}

// 	t.Run("Invalid ref: "+invalidRef.GetType(), func(t *testing.T) {
// 		err := r.Publish(t.Context(), invalidRef, nil)
// 		assert.Error(t, err)
// 		assert.Equal(t, "invalid object type: "+invalidRef.GetType(), err.Error())
// 	})
// }

// func TestList_InvalidQuery(t *testing.T) {
// 	r := &routing{}
// 	invalidQueries := []string{
// 		"",
// 		"/",
// 		"/agents",
// 		"/agents/agentX",
// 		"/skills/",
// 		"/locators",
// 		"skills/",
// 		"locators/",
// 	}

// 	for _, q := range invalidQueries {
// 		t.Run("Invalid query: "+q, func(t *testing.T) {
// 			_, err := r.List(t.Context(), q)
// 			assert.Error(t, err)
// 			assert.Equal(t, "invalid query: "+q, err.Error())
// 		})
// 	}
// }

func TestPublishList_ValidQuery(t *testing.T) {
	// Test data
	testAgent := &coretypes.Agent{
		Skills: []*coretypes.Skill{
			{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
		},
		Locators: []*coretypes.Locator{
			{Type: "type1", Url: "url1"},
		},
	}
	testRef := getObjectRef(testAgent)

	// testAgent2 := &coretypes.Agent{
	// 	Skills: []*coretypes.Skill{
	// 		{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
	// 		{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
	// 	},
	// }
	// 	testRef2 := getObjectRef(testAgent2)

	// 	validQueriesWithExpectedObjectRef := map[string][]*coretypes.ObjectRef{
	// 		// tests exact lookup for skills
	// 		"/skills/category1/class1": {
	// 			{
	// 				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
	// 				Digest: testRef.GetDigest(),
	// 			},
	// 			{
	// 				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
	// 				Digest: testRef2.GetDigest(),
	// 			},
	// 		},
	// 		// tests prefix based-lookup for skills
	// 		"/skills/category2": {
	// 			{
	// 				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
	// 				Digest: testRef2.GetDigest(),
	// 			},
	// 		},
	// 		// tests exact lookup for locators
	// 		"/locators/type1/url1": {
	// 			{
	// 				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
	// 				Digest: testRef.GetDigest(),
	// 			},
	// 		},
	// 		// tests prefix based-lookup for locators
	// 		"/locators/type1": {
	// 			{
	// 				Type:   coretypes.ObjectType_OBJECT_TYPE_AGENT.String(),
	// 				Digest: testRef.GetDigest(),
	// 			},
	// 		},
	// 	}

	// create demo network
	firstNode := newTestServer(t, t.Context(), nil)
	secondNode := newTestServer(t, t.Context(), firstNode.server.P2pAddrs())

	// wait for connection

	time.Sleep(2 * time.Second)
	<-firstNode.server.DHT().RefreshRoutingTable()
	<-secondNode.server.DHT().RefreshRoutingTable()

	// publish the key on first node and wait on the second
	//
	// TODO: when we receive announcement,
	// update the details about skils for that peer on the receiving node
	digestCID, err := testRef.GetCID()
	assert.NoError(t, err)
	err = secondNode.server.DHT().Provide(t.Context(), digestCID, true)
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)
	<-firstNode.server.DHT().RefreshRoutingTable()
	<-secondNode.server.DHT().RefreshRoutingTable()

	// // Publish first agent
	// err := r.Publish(t.Context(), testRef, testAgent)
	// assert.NoError(t, err)

	// // Publish second agent
	// err = r.Publish(t.Context(), testRef2, testAgent2)
	// assert.NoError(t, err)

	// for k, v := range validQueriesWithExpectedObjectRef {
	// 	t.Run("Valid query: "+k, func(t *testing.T) {
	// 		// list
	// 		refs, err := r.List(t.Context(), k)
	// 		assert.NoError(t, err)

	// 		// check if expected refs are present
	// 		assert.Len(t, refs, len(v))

	// 		for _, ref := range refs {
	// 			for _, r := range v {
	// 				if ref.GetDigest() == r.GetDigest() {
	// 					break
	// 				}
	// 			}
	// 		}
	// 	})
	// }
}

func TestPublish_Agent(t *testing.T) {
	// Test data
	testAgent := &coretypes.Agent{
		Skills: []*coretypes.Skill{
			{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
		},
		Locators: []*coretypes.Locator{
			{Type: "type1", Url: "url1"},
		},
	}
	testRef := getObjectRef(testAgent)

	// create demo network
	firstNode := newTestServer(t, t.Context(), nil)
	secondNode := newTestServer(t, t.Context(), firstNode.server.P2pAddrs())

	// wait for connection
	time.Sleep(2 * time.Second)
	<-firstNode.server.DHT().RefreshRoutingTable()
	<-secondNode.server.DHT().RefreshRoutingTable()

	// publish the key on second node and wait on the first
	digestCID, err := testRef.GetCID()
	assert.NoError(t, err)

	// announce the key
	err = secondNode.server.DHT().Provide(t.Context(), digestCID, true)
	assert.NoError(t, err)

	// wait for sync
	time.Sleep(2 * time.Second)
	<-firstNode.server.DHT().RefreshRoutingTable()
	<-secondNode.server.DHT().RefreshRoutingTable()

	// check on first
	found := false
	peerCh := firstNode.server.DHT().FindProvidersAsync(t.Context(), digestCID, 1)
	for peer := range peerCh {
		if peer.ID == secondNode.server.Host().ID() {
			found = true
			break
		}
	}
	assert.True(t, found)
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

func newTestServer(t *testing.T, ctx context.Context, bootPeers []string) *routing {
	t.Helper()

	// override interval for routing table refresh
	realInterval := refreshInterval
	refreshInterval = 1 * time.Second
	defer func() {
		refreshInterval = realInterval
	}()

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

	return r.(*routing)
}
