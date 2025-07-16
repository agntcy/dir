// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package routing

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/stretchr/testify/assert"
)

// Testing 2 nodes, A -> B
// stores and announces an agent.
// A discovers it retrieves the key metadata from B.
func TestHandler(t *testing.T) {
	// Test data
	testAgent := &coretypes.Agent{
		Agent: &objectsv1.Agent{
			Skills: []*objectsv1.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
			},
			Locators: []*objectsv1.Locator{
				{Type: "type1", Url: "url1"},
			},
		},
	}
	testRef := getObjectRef(testAgent)

	// create demo network
	firstNode := newTestServer(t, t.Context(), nil)
	secondNode := newTestServer(t, t.Context(), firstNode.remote.server.P2pAddrs())

	// wait for connection
	time.Sleep(2 * time.Second)
	<-firstNode.remote.server.DHT().RefreshRoutingTable()
	<-secondNode.remote.server.DHT().RefreshRoutingTable()

	// publish the key on second node and wait on the first
	digestCID, err := testRef.GetCID()
	assert.NoError(t, err)

	// push the data
	data, err := json.Marshal(testAgent)
	assert.NoError(t, err)
	_, err = secondNode.remote.storeAPI.Push(t.Context(), testRef, bytes.NewReader(data))
	assert.NoError(t, err)

	// announce the key
	err = secondNode.remote.server.DHT().Provide(t.Context(), digestCID, true)
	assert.NoError(t, err)

	// wait for sync
	time.Sleep(2 * time.Second)
	<-firstNode.remote.server.DHT().RefreshRoutingTable()
	<-secondNode.remote.server.DHT().RefreshRoutingTable()

	// check on first
	found := false

	peerCh := firstNode.remote.server.DHT().FindProvidersAsync(t.Context(), digestCID, 1)
	for peer := range peerCh {
		if peer.ID == secondNode.remote.server.Host().ID() {
			found = true

			break
		}
	}

	assert.True(t, found)
}
