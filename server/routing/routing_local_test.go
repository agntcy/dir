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
	"log/slog"
	"os"
	"testing"
	"time"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	routingtypes "github.com/agntcy/dir/api/routing/v1alpha2"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ipfsdatastore "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
)

func TestPublish_InvalidObject(t *testing.T) {
	r := &routeLocal{}

	t.Run("Invalid object", func(t *testing.T) {
		err := r.Publish(t.Context(), &coretypes.Object{
			Ref:   nil,
			Agent: nil,
		})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "invalid object reference")
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
		Agent: &objectsv1.Agent{},
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
			Agent: &objectsv1.Agent{
				Skills: []*objectsv1.Skill{
					{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
				},
			},
		}
		testAgent2 = &coretypes.Agent{
			Agent: &objectsv1.Agent{
				Skills: []*objectsv1.Skill{
					{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
					{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
				},
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
		Agent: testAgent.Agent,
	})
	assert.NoError(t, err)

	// Publish second agent
	err = r.Publish(t.Context(), &coretypes.Object{
		Ref:   testRef2,
		Agent: testAgent2.Agent,
	})
	assert.NoError(t, err)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// list
			refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
				LegacyListRequest: &routingtypes.LegacyListRequest{
					Labels: []string{k},
				},
			})
			assert.NoError(t, err)

			// Collect items from the channel
			var refs []*routingtypes.LegacyListResponse_Item
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
		Agent: testAgent2.Agent,
	})
	assert.NoError(t, err)

	// Try to list second agent
	refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
		LegacyListRequest: &routingtypes.LegacyListRequest{
			Labels: []string{"/skills/category2"},
		},
	})
	assert.NoError(t, err)

	// Collect items from the channel
	var refs []*routingtypes.LegacyListResponse_Item //nolint:prealloc
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
			Agent: &objectsv1.Agent{
				Skills: []*objectsv1.Skill{
					{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
					{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
				},
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
		Agent: testAgent.Agent,
	})
	assert.NoError(t, err)

	t.Run("Valid multi skill query", func(t *testing.T) {
		// list
		refsChan, err := r.List(t.Context(), &routingtypes.ListRequest{
			LegacyListRequest: &routingtypes.LegacyListRequest{
				Labels: []string{"/skills/category1/class1", "/skills/category2/class2"},
			},
		})
		assert.NoError(t, err)

		// Collect items from the channel
		var refs []*routingtypes.LegacyListResponse_Item
		for ref := range refsChan {
			refs = append(refs, ref)
		}

		// check if expected refs are present
		assert.Len(t, refs, 1)

		// check if expected ref is present
		assert.Equal(t, testRef.GetDigest(), refs[0].GetRecord().GetDigest())
	})
}

func newBadgerDatastore(b *testing.B) types.Datastore {
	b.Helper()

	dsOpts := []datastore.Option{
		datastore.WithFsProvider("/tmp/test-datastore"), // Use a temporary directory
	}

	dstore, err := datastore.New(dsOpts...)
	if err != nil {
		b.Fatalf("failed to create badger datastore: %v", err)
	}

	b.Cleanup(func() {
		_ = dstore.Close()
		_ = os.RemoveAll("/tmp/test-datastore")
	})

	return dstore
}

func newInMemoryDatastore(b *testing.B) types.Datastore {
	b.Helper()

	dstore, err := datastore.New()
	if err != nil {
		b.Fatalf("failed to create in-memory datastore: %v", err)
	}

	return dstore
}

func Benchmark_RouteLocal(b *testing.B) {
	store := newMockStore()
	badgerDatastore := newBadgerDatastore(b)
	inMemoryDatastore := newInMemoryDatastore(b)
	localLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	badgerRouter := newLocal(store, badgerDatastore)
	inMemoryRouter := newLocal(store, inMemoryDatastore)

	agent := &coretypes.Agent{
		Agent: &objectsv1.Agent{
			Skills: []*objectsv1.Skill{
				{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
			},
		},
	}
	ref := getObjectRef(agent)
	object := &coretypes.Object{Ref: ref, Agent: agent.Agent}

	agentData, err := json.Marshal(agent)
	assert.NoError(b, err)

	_, err = store.Push(b.Context(), ref, bytes.NewReader(agentData))
	assert.NoError(b, err)

	b.Run("Badger DB Publish and Unpublish", func(b *testing.B) {
		for b.Loop() {
			_ = badgerRouter.Publish(b.Context(), object)
			err := badgerRouter.Unpublish(b.Context(), object)
			assert.NoError(b, err)
		}
	})

	b.Run("Badger DB List", func(b *testing.B) {
		_ = badgerRouter.Publish(b.Context(), object)
		for b.Loop() {
			_, err := badgerRouter.List(b.Context(), &routingtypes.ListRequest{
				LegacyListRequest: &routingtypes.LegacyListRequest{
					Labels: []string{"/skills/category1/class1"},
				},
			})
			assert.NoError(b, err)
		}
	})

	b.Run("In memory DB Publish and Unpublish", func(b *testing.B) {
		for b.Loop() {
			_ = inMemoryRouter.Publish(b.Context(), object)
			err := inMemoryRouter.Unpublish(b.Context(), object)
			assert.NoError(b, err)
		}
	})

	b.Run("In memory DB List", func(b *testing.B) {
		_ = inMemoryRouter.Publish(b.Context(), object)
		for b.Loop() {
			_, err := inMemoryRouter.List(b.Context(), &routingtypes.ListRequest{
				LegacyListRequest: &routingtypes.LegacyListRequest{
					Labels: []string{"/skills/category1/class1"},
				},
			})
			assert.NoError(b, err)
		}
	})

	_ = badgerDatastore.Delete(b.Context(), ipfsdatastore.NewKey("/"))   // Delete all keys
	_ = inMemoryDatastore.Delete(b.Context(), ipfsdatastore.NewKey("/")) // Delete all keys
	localLogger = logging.Logger("routing/local")
}
