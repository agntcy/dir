// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint,wsl
package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestPublish_InvalidObject(t *testing.T) {
	r := &routeLocal{}

	t.Run("Invalid object", func(t *testing.T) {
		err := r.Publish(t.Context(), &coretypes.Object{
			Ref:   nil,
			Agent: nil,
		}, true)

		assert.Error(t, err)
		assert.Equal(t, "invalid object reference: <nil>", err.Error())
	})
}

type mockStore struct {
	data map[string]*coretypes.Object
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string]*coretypes.Object),
	}
}

func (m *mockStore) Push(_ context.Context, ref *coretypes.ObjectRef, contents io.Reader) (*coretypes.ObjectRef, error) {
	b, err := io.ReadAll(contents)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	m.data[ref.GetDigest()] = &coretypes.Object{
		Ref:   ref,
		Agent: &coretypes.Agent{},
		Data:  b,
	}

	return ref, nil
}

func (m *mockStore) Lookup(_ context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	if obj, exists := m.data[ref.GetDigest()]; exists {
		return obj.GetRef(), nil
	}

	return nil, errors.New("test object not found")
}

func (m *mockStore) Pull(_ context.Context, ref *coretypes.ObjectRef) (io.ReadCloser, error) {
	if obj, exists := m.data[ref.GetDigest()]; exists {
		return io.NopCloser(bytes.NewReader(obj.GetData())), nil
	}

	return nil, errors.New("test object not found")
}

func (m *mockStore) Delete(_ context.Context, ref *coretypes.ObjectRef) error {
	delete(m.data, ref.GetDigest())

	return nil
}

func TestPublishList_ValidSingleSkillQuery(t *testing.T) {
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
	mainNode := newTestServer(t, t.Context(), nil)
	r := newTestServer(t, t.Context(), mainNode.remote.server.P2pAddrs())

	// wait for connection
	<-mainNode.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Mock store
	mockstore := newMockStore()
	r.local.store = mockstore

	agentData, err := json.Marshal(testAgent)
	assert.NoError(t, err)

	_, err = r.local.store.Push(t.Context(), testRef, bytes.NewReader(agentData))
	assert.NoError(t, err)

	agentData2, err := json.Marshal(testAgent2)
	assert.NoError(t, err)

	_, err = r.local.store.Push(t.Context(), testRef2, bytes.NewReader(agentData2))
	assert.NoError(t, err)

	// Publish first agent
	err = r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef,
		Agent: testAgent,
	}, false)
	assert.NoError(t, err)

	// Publish second agent
	err = r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef2,
		Agent: testAgent2,
	}, false)
	assert.NoError(t, err)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// list
			refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
				Network: toPtr(false),
				Labels:  []string{k},
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

	// Unpublish second agent
	err = r.Unpublish(t.Context(), &coretypes.Object{
		Ref:   testRef2,
		Agent: testAgent2,
	}, false)
	assert.NoError(t, err)

	// Try to list second agent
	refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
		Network: toPtr(false),
		Labels:  []string{"/skills/category2"},
	})
	assert.NoError(t, err)

	// Collect items from the channel
	var refs []*routingtypes.ListResponse_Item //nolint:prealloc
	for ref := range refsChan {
		refs = append(refs, ref)
	}

	// check no refs are present
	assert.Len(t, refs, 0)
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
	mainNode := newTestServer(t, t.Context(), nil)
	r := newTestServer(t, t.Context(), mainNode.remote.server.P2pAddrs())

	// wait for connection
	<-mainNode.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Mock store
	mockstore := newMockStore()
	r.local.store = mockstore

	agentData, err := json.Marshal(testAgent)
	assert.NoError(t, err)

	_, err = r.local.store.Push(t.Context(), testRef, bytes.NewReader(agentData))
	assert.NoError(t, err)

	// Publish first agent
	err = r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef,
		Agent: testAgent,
	}, true)
	assert.NoError(t, err)

	t.Run("Valid multi skill query", func(t *testing.T) {
		// list
		refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
			Network: toPtr(false),
			Labels:  []string{"/skills/category1/class1", "/skills/category2/class2"},
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
