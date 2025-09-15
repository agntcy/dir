// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package routing

import (
	"testing"
	"time"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
)

// Testing 2 nodes, A -> B
// stores and announces an record.
// A discovers it retrieves the key metadata from B.
func TestHandler(t *testing.T) {
	// Test data
	testRecord := corev1.New(&typesv1alpha0.Record{
		Name: "test-handler-agent",
		Skills: []*typesv1alpha0.Skill{
			{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
		},
		Locators: []*typesv1alpha0.Locator{
			{Type: "type1", Url: "url1"},
		},
	})
	testRef := &corev1.RecordRef{Cid: testRecord.GetCid()}

	// create demo network
	firstNode := newTestServer(t, t.Context(), nil)
	secondNode := newTestServer(t, t.Context(), firstNode.remote.server.P2pAddrs())

	// wait for connection
	time.Sleep(2 * time.Second)
	<-firstNode.remote.server.DHT().RefreshRoutingTable()
	<-secondNode.remote.server.DHT().RefreshRoutingTable()

	// publish the key on second node and wait on the first
	cidStr := testRef.GetCid()
	decodedCID, err := cid.Decode(cidStr)
	assert.NoError(t, err)

	// push the data
	_, err = secondNode.remote.storeAPI.Push(t.Context(), testRecord)
	assert.NoError(t, err)

	// announce the key
	err = secondNode.remote.server.DHT().Provide(t.Context(), decodedCID, true)
	assert.NoError(t, err)

	// wait for sync
	time.Sleep(2 * time.Second)
	<-firstNode.remote.server.DHT().RefreshRoutingTable()
	<-secondNode.remote.server.DHT().RefreshRoutingTable()

	// check on first
	found := false

	peerCh := firstNode.remote.server.DHT().FindProvidersAsync(t.Context(), decodedCID, 1)
	for peer := range peerCh {
		if peer.ID == secondNode.remote.server.Host().ID() {
			found = true

			break
		}
	}

	assert.True(t, found)
}
