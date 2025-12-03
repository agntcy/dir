// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	storev1 "github.com/agntcy/dir/api/store/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/stretchr/testify/assert"
)

// TODO: this should be configurable to unified Storage API test flow.
var (
	// test config.
	testConfig = ociconfig.Config{
		LocalDir:        os.TempDir(),                         // used for local test/bench
		RegistryAddress: "localhost:5000",                     // used for remote test/bench
		RepositoryName:  "test-store",                         // used for remote test/bench
		AuthConfig:      ociconfig.AuthConfig{Insecure: true}, // used for remote test/bench
	}
	runLocal = true
	// TODO: this may blow quickly when doing rapid benchmarking if not tested against fresh OCI instance.
	runRemote = false

	// common test.
	testCtx = context.Background()
)

func TestStore(t *testing.T) {
	t.Run("PushPullSimpleRecord", TestPushPullSimpleRecord)
	t.Run("PushRecordWithLinks", TestPushRecordWithLinks)
	t.Run("PushRecordWithParent", TestPushRecordWithParent)
	t.Run("PushComplexDAG", TestPushComplexDAG)
	t.Run("LookupRecord", TestLookupRecord)
}

// TestPushPullSimpleRecord tests pushing and pulling a basic record without links or parent.
func TestPushPullSimpleRecord(t *testing.T) {
	store := loadLocalStore(t)

	objectType := "test.record.v1"
	createdAt := "2025-12-03T10:00:00Z"

	// Create a simple object
	obj := &storev1.Object{
		Type:      &objectType,
		CreatedAt: &createdAt,
		Annotations: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	data := `{"name": "test-agent", "version": "1.0.0"}`
	dataReader := io.NopCloser(strings.NewReader(data))

	// Push object
	ref, err := store.Push(testCtx, obj, dataReader)
	assert.NoError(t, err, "Push should succeed")
	assert.NotNil(t, ref, "ObjectRef should not be nil")
	assert.NotEmpty(t, ref.Cid, "CID should not be empty")

	// Pull object back
	pulledObj, pulledDataReader, err := store.Pull(testCtx, ref)
	assert.NoError(t, err, "Pull should succeed")
	assert.NotNil(t, pulledObj, "Pulled object should not be nil")
	assert.NotNil(t, pulledDataReader, "Pulled data reader should not be nil")
	defer pulledDataReader.Close()

	// Verify metadata
	assert.Equal(t, *obj.Type, *pulledObj.Type, "Type should match")
	assert.Equal(t, *obj.CreatedAt, *pulledObj.CreatedAt, "CreatedAt should match")
	assert.Equal(t, obj.Annotations, pulledObj.Annotations, "Annotations should match")
	assert.Equal(t, uint64(len(data)), pulledObj.Size, "Size should match")

	// Verify data content
	pulledData, err := io.ReadAll(pulledDataReader)
	assert.NoError(t, err, "Reading pulled data should succeed")
	assert.Equal(t, data, string(pulledData), "Data content should match")
}

// TestPushRecordWithLinks tests creating a DAG with forward references (links).
func TestPushRecordWithLinks(t *testing.T) {
	store := loadLocalStore(t)

	leafType := "test.leaf.v1"
	leaf1CreatedAt := "2025-12-03T10:00:00Z"
	leaf2CreatedAt := "2025-12-03T10:01:00Z"

	// Create leaf objects first (bottom-up DAG creation)
	leaf1Obj := &storev1.Object{
		Type:      &leafType,
		CreatedAt: &leaf1CreatedAt,
	}
	leaf1Data := `{"data": "leaf1"}`
	leaf1Ref, err := store.Push(testCtx, leaf1Obj, io.NopCloser(strings.NewReader(leaf1Data)))
	assert.NoError(t, err, "Leaf1 push should succeed")

	leaf2Obj := &storev1.Object{
		Type:      &leafType,
		CreatedAt: &leaf2CreatedAt,
	}
	leaf2Data := `{"data": "leaf2"}`
	leaf2Ref, err := store.Push(testCtx, leaf2Obj, io.NopCloser(strings.NewReader(leaf2Data)))
	assert.NoError(t, err, "Leaf2 push should succeed")

	parentType := "test.parent.v1"
	parentCreatedAt := "2025-12-03T10:02:00Z"

	// Create parent object with links to leaves
	parentObj := &storev1.Object{
		Type:      &parentType,
		CreatedAt: &parentCreatedAt,
		Links: []*storev1.ObjectRef{
			{Cid: leaf1Ref.Cid},
			{Cid: leaf2Ref.Cid},
		},
	}
	parentData := `{"name": "parent-record"}`
	parentRef, err := store.Push(testCtx, parentObj, io.NopCloser(strings.NewReader(parentData)))
	assert.NoError(t, err, "Parent push should succeed")

	// Pull parent and verify links
	pulledParentObj, pulledDataReader, err := store.Pull(testCtx, parentRef)
	assert.NoError(t, err, "Parent pull should succeed")
	defer pulledDataReader.Close()

	assert.Len(t, pulledParentObj.Links, 2, "Parent should have 2 links")
	assert.Equal(t, leaf1Ref.Cid, pulledParentObj.Links[0].Cid, "First link should match leaf1")
	assert.Equal(t, leaf2Ref.Cid, pulledParentObj.Links[1].Cid, "Second link should match leaf2")

	// Verify we can still pull the leaves
	_, leaf1Reader, err := store.Pull(testCtx, leaf1Ref)
	assert.NoError(t, err, "Leaf1 should still be pullable")
	leaf1Reader.Close()

	_, leaf2Reader, err := store.Pull(testCtx, leaf2Ref)
	assert.NoError(t, err, "Leaf2 should still be pullable")
	leaf2Reader.Close()
}

// testPushRecordWithParent tests creating records with reverse references (OCI Referrers).
func TestPushRecordWithParent(t *testing.T) {
	store := loadLocalStore(t)

	parentType := "test.parent.v1"
	parentCreatedAt := "2025-12-03T10:00:00Z"

	// Create parent object first
	parentObj := &storev1.Object{
		Type:      &parentType,
		CreatedAt: &parentCreatedAt,
	}
	parentData := `{"name": "parent"}`
	parentRef, err := store.Push(testCtx, parentObj, io.NopCloser(strings.NewReader(parentData)))
	assert.NoError(t, err, "Parent push should succeed")

	childType := "test.child.v1"
	childCreatedAt := "2025-12-03T10:01:00Z"

	// Create child object with parent reference (reverse ref)
	childObj := &storev1.Object{
		Type:      &childType,
		CreatedAt: &childCreatedAt,
		Parent:    &storev1.ObjectRef{Cid: parentRef.Cid},
	}
	childData := `{"name": "child"}`
	childRef, err := store.Push(testCtx, childObj, io.NopCloser(strings.NewReader(childData)))
	assert.NoError(t, err, "Child push should succeed")

	// Pull child and verify parent reference
	pulledChildObj, childReader, err := store.Pull(testCtx, childRef)
	assert.NoError(t, err, "Child pull should succeed")
	defer childReader.Close()

	assert.NotNil(t, pulledChildObj.Parent, "Child should have parent reference")
	assert.Equal(t, parentRef.Cid, pulledChildObj.Parent.Cid, "Parent CID should match")
}

// testPushComplexDAG tests creating a multi-level DAG with both links and parents.
func TestPushComplexDAG(t *testing.T) {
	store := loadLocalStore(t)

	leafType := "test.data.v1"
	leaf1CreatedAt := "2025-12-03T10:00:00Z"
	leaf2CreatedAt := "2025-12-03T10:00:01Z"

	// Level 3: Leaf nodes (data)
	leaf1Obj := &storev1.Object{
		Type:      &leafType,
		CreatedAt: &leaf1CreatedAt,
	}
	leaf1Ref, err := store.Push(testCtx, leaf1Obj, io.NopCloser(strings.NewReader(`{"value": 1}`)))
	assert.NoError(t, err)

	leaf2Obj := &storev1.Object{
		Type:      &leafType,
		CreatedAt: &leaf2CreatedAt,
	}
	leaf2Ref, err := store.Push(testCtx, leaf2Obj, io.NopCloser(strings.NewReader(`{"value": 2}`)))
	assert.NoError(t, err)

	midType := "test.intermediate.v1"
	mid1CreatedAt := "2025-12-03T10:01:00Z"
	mid2CreatedAt := "2025-12-03T10:01:01Z"

	// Level 2: Middle nodes (link to leaves)
	mid1Obj := &storev1.Object{
		Type:      &midType,
		CreatedAt: &mid1CreatedAt,
		Links: []*storev1.ObjectRef{
			{Cid: leaf1Ref.Cid},
		},
	}
	mid1Ref, err := store.Push(testCtx, mid1Obj, io.NopCloser(strings.NewReader(`{"layer": "middle1"}`)))
	assert.NoError(t, err)

	mid2Obj := &storev1.Object{
		Type:      &midType,
		CreatedAt: &mid2CreatedAt,
		Links: []*storev1.ObjectRef{
			{Cid: leaf2Ref.Cid},
		},
	}
	mid2Ref, err := store.Push(testCtx, mid2Obj, io.NopCloser(strings.NewReader(`{"layer": "middle2"}`)))
	assert.NoError(t, err)

	rootType := "test.root.v1"
	rootCreatedAt := "2025-12-03T10:02:00Z"

	// Level 1: Root node (links to middle nodes)
	rootObj := &storev1.Object{
		Type:      &rootType,
		CreatedAt: &rootCreatedAt,
		Links: []*storev1.ObjectRef{
			{Cid: mid1Ref.Cid},
			{Cid: mid2Ref.Cid},
		},
		Annotations: map[string]string{
			"dag.level": "root",
			"dag.depth": "3",
		},
	}
	rootRef, err := store.Push(testCtx, rootObj, io.NopCloser(strings.NewReader(`{"layer": "root"}`)))
	assert.NoError(t, err)

	// Verify DAG structure by traversing from root
	rootPulled, _, err := store.Pull(testCtx, rootRef)
	assert.NoError(t, err)
	assert.Len(t, rootPulled.Links, 2, "Root should have 2 children")
	assert.Equal(t, "3", rootPulled.Annotations["dag.depth"])

	// Verify middle layer
	mid1Pulled, _, err := store.Pull(testCtx, &storev1.ObjectRef{Cid: mid1Ref.Cid})
	assert.NoError(t, err)
	assert.Len(t, mid1Pulled.Links, 1, "Mid1 should have 1 child")
	assert.Equal(t, leaf1Ref.Cid, mid1Pulled.Links[0].Cid)

	mid2Pulled, _, err := store.Pull(testCtx, &storev1.ObjectRef{Cid: mid2Ref.Cid})
	assert.NoError(t, err)
	assert.Len(t, mid2Pulled.Links, 1, "Mid2 should have 1 child")
	assert.Equal(t, leaf2Ref.Cid, mid2Pulled.Links[0].Cid)

	// Verify leaf nodes
	leaf1Pulled, _, err := store.Pull(testCtx, leaf1Ref)
	assert.NoError(t, err)
	assert.Len(t, leaf1Pulled.Links, 0, "Leaf1 should have no children")

	leaf2Pulled, _, err := store.Pull(testCtx, leaf2Ref)
	assert.NoError(t, err)
	assert.Len(t, leaf2Pulled.Links, 0, "Leaf2 should have no children")
}

// testLookupRecord tests the Lookup operation for retrieving metadata only.
func TestLookupRecord(t *testing.T) {
	store := loadLocalStore(t)

	lookupType := "test.lookup.v1"
	createdAt := "2025-12-03T10:00:00Z"

	// Push an object
	obj := &storev1.Object{
		Type:      &lookupType,
		CreatedAt: &createdAt,
		Annotations: map[string]string{
			"test": "lookup",
		},
	}
	data := `{"test": "data"}`
	ref, err := store.Push(testCtx, obj, io.NopCloser(strings.NewReader(data)))
	assert.NoError(t, err)

	// Lookup metadata
	lookedUpObj, err := store.Lookup(testCtx, ref)
	assert.NoError(t, err, "Lookup should succeed")
	assert.NotNil(t, lookedUpObj, "Looked up object should not be nil")
	assert.Equal(t, *obj.Type, *lookedUpObj.Type)
	assert.Equal(t, *obj.CreatedAt, *lookedUpObj.CreatedAt)
	assert.Equal(t, obj.Annotations, lookedUpObj.Annotations)
}

func loadLocalStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create local
	// store, err := New(ociconfig.Config{LocalDir: "./testdata/oci_local_store"})
	// assert.NoErrorf(t, err, "failed to create local store")

	return loadRemoteStore(t)
}

func loadRemoteStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create local
	store, err := New(ociconfig.Config{
		RegistryAddress: "localhost:5000",
		RepositoryName:  "test-store",
		AuthConfig:      ociconfig.AuthConfig{Insecure: true},
	})
	assert.NoErrorf(t, err, "failed to create local store")

	return store
}

// Benchmark functions

// BenchmarkPushSimpleObjects benchmarks pushing simple objects without links.
func BenchmarkPushSimpleObjects(b *testing.B) {
	store := loadBenchStore(b)

	objectType := "benchmark.object.v1"
	createdAt := "2025-12-03T10:00:00Z"

	sampleData := `{"name": "benchmark-object", "version": "1.0.0", "description": "Sample benchmark data"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := &storev1.Object{
			Type:      &objectType,
			CreatedAt: &createdAt,
			Annotations: map[string]string{
				"benchmark": "true",
				"iteration": string(rune(i)),
			},
		}

		_, err := store.Push(testCtx, obj, io.NopCloser(strings.NewReader(sampleData)))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}
	}
}

// BenchmarkPushObjectsWithLinks benchmarks pushing objects with forward references.
func BenchmarkPushObjectsWithLinks(b *testing.B) {
	store := loadBenchStore(b)

	leafType := "benchmark.leaf.v1"
	parentType := "benchmark.parent.v1"
	createdAt := "2025-12-03T10:00:00Z"

	// Create leaf objects once for reuse
	leafData := `{"data": "leaf-content"}`
	leafObj := &storev1.Object{
		Type:      &leafType,
		CreatedAt: &createdAt,
	}
	leafRef1, err := store.Push(testCtx, leafObj, io.NopCloser(strings.NewReader(leafData)))
	if err != nil {
		b.Fatalf("Failed to create leaf1: %v", err)
	}
	leafRef2, err := store.Push(testCtx, leafObj, io.NopCloser(strings.NewReader(leafData)))
	if err != nil {
		b.Fatalf("Failed to create leaf2: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parentObj := &storev1.Object{
			Type:      &parentType,
			CreatedAt: &createdAt,
			Links: []*storev1.ObjectRef{
				{Cid: leafRef1.Cid},
				{Cid: leafRef2.Cid},
			},
		}

		parentData := `{"name": "parent-with-links"}`
		_, err := store.Push(testCtx, parentObj, io.NopCloser(strings.NewReader(parentData)))
		if err != nil {
			b.Fatalf("Push with links failed: %v", err)
		}
	}
}

// BenchmarkPushPullCycle benchmarks a complete push-pull cycle.
func BenchmarkPushPullCycle(b *testing.B) {
	store := loadBenchStore(b)

	objectType := "benchmark.cycle.v1"
	createdAt := "2025-12-03T10:00:00Z"
	sampleData := `{"test": "benchmark-cycle-data"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := &storev1.Object{
			Type:      &objectType,
			CreatedAt: &createdAt,
		}

		// Push
		ref, err := store.Push(testCtx, obj, io.NopCloser(strings.NewReader(sampleData)))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}

		// Pull
		_, reader, err := store.Pull(testCtx, ref)
		if err != nil {
			b.Fatalf("Pull failed: %v", err)
		}
		reader.Close()
	}
}

// BenchmarkLookup benchmarks metadata lookup operations.
func BenchmarkLookup(b *testing.B) {
	store := loadBenchStore(b)

	objectType := "benchmark.lookup.v1"
	createdAt := "2025-12-03T10:00:00Z"
	sampleData := `{"test": "lookup-data"}`

	// Push objects first
	refs := make([]*storev1.ObjectRef, 100)
	for i := 0; i < 100; i++ {
		obj := &storev1.Object{
			Type:      &objectType,
			CreatedAt: &createdAt,
		}
		ref, err := store.Push(testCtx, obj, io.NopCloser(strings.NewReader(sampleData)))
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
		refs[i] = ref
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ref := refs[i%100]
		_, err := store.Lookup(testCtx, ref)
		if err != nil {
			b.Fatalf("Lookup failed: %v", err)
		}
	}
}

// BenchmarkPushLargeData benchmarks pushing objects with larger data payloads.
func BenchmarkPushLargeData(b *testing.B) {
	store := loadBenchStore(b)

	objectType := "benchmark.large.v1"
	createdAt := "2025-12-03T10:00:00Z"

	// Create 100KB sample data
	largeData := strings.Repeat(`{"key": "value", "data": "sample content"}`, 2500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		obj := &storev1.Object{
			Type:      &objectType,
			CreatedAt: &createdAt,
		}

		_, err := store.Push(testCtx, obj, io.NopCloser(strings.NewReader(largeData)))
		if err != nil {
			b.Fatalf("Push large data failed: %v", err)
		}
	}
}

// loadBenchStore creates a store instance for benchmarking.
func loadBenchStore(b *testing.B) types.StoreAPI {
	b.Helper()

	return loadRemoteStore(&testing.T{})
}
