// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint:testifylint,wsl
package routing

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	objectsv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	routingv1 "github.com/agntcy/dir/api/routing/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ipfsdatastore "github.com/ipfs/go-datastore"
	"github.com/stretchr/testify/assert"
)

const testPeerID = "test-peer-id"

func TestPublish_InvalidObject(t *testing.T) {
	r := &routeLocal{localPeerID: testPeerID}

	t.Run("Invalid object", func(t *testing.T) {
		err := r.Publish(t.Context(), nil, &corev1.Record{})

		assert.Error(t, err)
		assert.ErrorContains(t, err, "record reference is required")
	})
}

type mockStore struct {
	data map[string]*corev1.Record
}

func newMockStore() *mockStore {
	return &mockStore{
		data: make(map[string]*corev1.Record),
	}
}

func (m *mockStore) Push(_ context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	cid := record.GetCid()
	if cid == "" {
		return nil, errors.New("record CID is required")
	}

	m.data[cid] = record

	return &corev1.RecordRef{Cid: cid}, nil
}

func (m *mockStore) Lookup(_ context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	if _, exists := m.data[ref.GetCid()]; exists {
		return &corev1.RecordMeta{
			Cid: ref.GetCid(),
		}, nil
	}

	return nil, errors.New("test object not found")
}

func (m *mockStore) Pull(_ context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	if record, exists := m.data[ref.GetCid()]; exists {
		return record, nil
	}

	return nil, errors.New("test object not found")
}

func (m *mockStore) Delete(_ context.Context, ref *corev1.RecordRef) error {
	delete(m.data, ref.GetCid())

	return nil
}

func TestPublishList_ValidSingleSkillQuery(t *testing.T) {
	var (
		testRecord = &corev1.Record{
			Data: &corev1.Record_V1{
				V1: &objectsv1.Agent{
					Name: "test-agent-1",
					Skills: []*objectsv1.Skill{
						{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
					},
				},
			},
		}
		testRecord2 = &corev1.Record{
			Data: &corev1.Record_V1{
				V1: &objectsv1.Agent{
					Name: "test-agent-2",
					Skills: []*objectsv1.Skill{
						{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
						{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
					},
				},
			},
		}

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
	mainNode := newTestServer(t, t.Context(), nil)
	r := newTestServer(t, t.Context(), mainNode.remote.server.P2pAddrs())

	// wait for connection
	<-mainNode.remote.server.DHT().RefreshRoutingTable()
	time.Sleep(1 * time.Second)

	// Mock store
	mockstore := newMockStore()
	r.local.store = mockstore

	_, err := r.local.store.Push(t.Context(), testRecord)
	assert.NoError(t, err)

	_, err = r.local.store.Push(t.Context(), testRecord2)
	assert.NoError(t, err)

	// Publish first record
	err = r.Publish(t.Context(), testRef, testRecord)
	assert.NoError(t, err)

	// Publish second record
	err = r.Publish(t.Context(), testRef2, testRecord2)
	assert.NoError(t, err)

	for k, v := range validQueriesWithExpectedObjectRef {
		t.Run("Valid query: "+k, func(t *testing.T) {
			// Convert label to RecordQuery
			var queries []*routingv1.RecordQuery
			if strings.HasPrefix(k, "/skills/") {
				skillName := strings.TrimPrefix(k, "/skills/")
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
	err = r.Unpublish(t.Context(), testRef2, testRecord2)
	assert.NoError(t, err)

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
	var refs []*routingv1.ListResponse //nolint:prealloc
	for ref := range refsChan {
		refs = append(refs, ref)
	}

	// check no refs are present
	assert.Len(t, refs, 0)
}

func TestPublishList_ValidMultiSkillQuery(t *testing.T) {
	// Test data
	var (
		testRecord = &corev1.Record{
			Data: &corev1.Record_V1{
				V1: &objectsv1.Agent{
					Name: "test-agent-multi",
					Skills: []*objectsv1.Skill{
						{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
						{CategoryName: toPtr("category2"), ClassName: toPtr("class2")},
					},
				},
			},
		}

		testRef = &corev1.RecordRef{Cid: testRecord.GetCid()}
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

	_, err := r.local.store.Push(t.Context(), testRecord)
	assert.NoError(t, err)

	// Publish first record
	err = r.Publish(t.Context(), testRef, testRecord)
	assert.NoError(t, err)

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

	badgerRouter := newLocal(store, badgerDatastore, testPeerID)
	inMemoryRouter := newLocal(store, inMemoryDatastore, testPeerID)

	record := &corev1.Record{
		Data: &corev1.Record_V1{
			V1: &objectsv1.Agent{
				Name: "bench-agent",
				Skills: []*objectsv1.Skill{
					{CategoryName: toPtr("category1"), ClassName: toPtr("class1")},
				},
			},
		},
	}
	ref := &corev1.RecordRef{Cid: record.GetCid()}

	_, err := store.Push(b.Context(), record)
	assert.NoError(b, err)

	b.Run("Badger DB Publish and Unpublish", func(b *testing.B) {
		for b.Loop() {
			_ = badgerRouter.Publish(b.Context(), ref, record)
			err := badgerRouter.Unpublish(b.Context(), ref, record)
			assert.NoError(b, err)
		}
	})

	b.Run("Badger DB List", func(b *testing.B) {
		_ = badgerRouter.Publish(b.Context(), ref, record)
		for b.Loop() {
			_, err := badgerRouter.List(b.Context(), &routingv1.ListRequest{
				Queries: []*routingv1.RecordQuery{
					{
						Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
						Value: "category1/class1",
					},
				},
			})
			assert.NoError(b, err)
		}
	})

	b.Run("In memory DB Publish and Unpublish", func(b *testing.B) {
		for b.Loop() {
			_ = inMemoryRouter.Publish(b.Context(), ref, record)
			err := inMemoryRouter.Unpublish(b.Context(), ref, record)
			assert.NoError(b, err)
		}
	})

	b.Run("In memory DB List", func(b *testing.B) {
		_ = inMemoryRouter.Publish(b.Context(), ref, record)
		for b.Loop() {
			_, err := inMemoryRouter.List(b.Context(), &routingv1.ListRequest{
				Queries: []*routingv1.RecordQuery{
					{
						Type:  routingv1.RecordQueryType_RECORD_QUERY_TYPE_SKILL,
						Value: "category1/class1",
					},
				},
			})
			assert.NoError(b, err)
		}
	})

	_ = badgerDatastore.Delete(b.Context(), ipfsdatastore.NewKey("/"))   // Delete all keys
	_ = inMemoryDatastore.Delete(b.Context(), ipfsdatastore.NewKey("/")) // Delete all keys
	localLogger = logging.Logger("routing/local")
}
