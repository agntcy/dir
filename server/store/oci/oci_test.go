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

	corev1 "github.com/agntcy/dir/api/core/v1"
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
	t.Run("DeleteRecord", TestDeleteRecord)
	t.Run("DeleteNonExistentRecord", TestDeleteNonExistentRecord)
}

// TestPushPullSimpleRecord tests pushing and pulling a basic record without links or parent.
func TestPushPullSimpleRecord(t *testing.T) {
	store := loadLocalStore(t)

	// Create a simple record
	meta := &corev1.RecordMeta{
		Cid:       "bafytest123",
		Type:      "test.record.v1",
		CreatedAt: "2025-12-03T10:00:00Z",
		Size:      100,
		Annotations: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	data := `{"name": "test-agent", "version": "1.0.0"}`
	dataReader := io.NopCloser(strings.NewReader(data))

	// Push record
	ref, err := store.Push(testCtx, meta, dataReader)
	assert.NoError(t, err, "Push should succeed")
	assert.NotNil(t, ref, "RecordRef should not be nil")
	assert.NotEmpty(t, ref.Cid, "CID should not be empty")

	// Pull record back
	pulledMeta, pulledDataReader, err := store.Pull(testCtx, ref)
	assert.NoError(t, err, "Pull should succeed")
	assert.NotNil(t, pulledMeta, "Pulled metadata should not be nil")
	assert.NotNil(t, pulledDataReader, "Pulled data reader should not be nil")
	defer pulledDataReader.Close()

	// Verify metadata
	assert.Equal(t, meta.Type, pulledMeta.Type, "Type should match")
	assert.Equal(t, meta.CreatedAt, pulledMeta.CreatedAt, "CreatedAt should match")
	assert.Equal(t, meta.Annotations, pulledMeta.Annotations, "Annotations should match")

	// Verify data content
	pulledData, err := io.ReadAll(pulledDataReader)
	assert.NoError(t, err, "Reading pulled data should succeed")
	assert.Equal(t, data, string(pulledData), "Data content should match")
}

// TestPushRecordWithLinks tests creating a DAG with forward references (links).
func TestPushRecordWithLinks(t *testing.T) {
	store := loadLocalStore(t)

	// Create leaf records first (bottom-up DAG creation)
	leaf1Meta := &corev1.RecordMeta{
		Cid:       "bafyleaf1",
		Type:      "test.leaf.v1",
		CreatedAt: "2025-12-03T10:00:00Z",
		Size:      50,
	}
	leaf1Data := `{"data": "leaf1"}`
	leaf1Ref, err := store.Push(testCtx, leaf1Meta, io.NopCloser(strings.NewReader(leaf1Data)))
	assert.NoError(t, err, "Leaf1 push should succeed")

	leaf2Meta := &corev1.RecordMeta{
		Cid:       "bafyleaf2",
		Type:      "test.leaf.v1",
		CreatedAt: "2025-12-03T10:01:00Z",
		Size:      50,
	}
	leaf2Data := `{"data": "leaf2"}`
	leaf2Ref, err := store.Push(testCtx, leaf2Meta, io.NopCloser(strings.NewReader(leaf2Data)))
	assert.NoError(t, err, "Leaf2 push should succeed")

	// Create parent record with links to leaves
	parentMeta := &corev1.RecordMeta{
		Cid:       "bafyparent",
		Type:      "test.parent.v1",
		CreatedAt: "2025-12-03T10:02:00Z",
		Size:      100,
		Links: []*corev1.RecordRef{
			{Cid: leaf1Ref.Cid},
			{Cid: leaf2Ref.Cid},
		},
	}
	parentData := `{"name": "parent-record"}`
	parentRef, err := store.Push(testCtx, parentMeta, io.NopCloser(strings.NewReader(parentData)))
	assert.NoError(t, err, "Parent push should succeed")

	// Pull parent and verify links
	pulledParentMeta, pulledDataReader, err := store.Pull(testCtx, parentRef)
	assert.NoError(t, err, "Parent pull should succeed")
	defer pulledDataReader.Close()

	assert.Len(t, pulledParentMeta.Links, 2, "Parent should have 2 links")
	assert.Equal(t, leaf1Ref.Cid, pulledParentMeta.Links[0].Cid, "First link should match leaf1")
	assert.Equal(t, leaf2Ref.Cid, pulledParentMeta.Links[1].Cid, "Second link should match leaf2")

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

	// Create parent record first
	parentMeta := &corev1.RecordMeta{
		Cid:       "bafyparent123",
		Type:      "test.parent.v1",
		CreatedAt: "2025-12-03T10:00:00Z",
		Size:      100,
	}
	parentData := `{"name": "parent"}`
	parentRef, err := store.Push(testCtx, parentMeta, io.NopCloser(strings.NewReader(parentData)))
	assert.NoError(t, err, "Parent push should succeed")

	// Create child record with parent reference (reverse ref)
	childMeta := &corev1.RecordMeta{
		Cid:       "bafychild123",
		Type:      "test.child.v1",
		CreatedAt: "2025-12-03T10:01:00Z",
		Size:      50,
		Parent:    &corev1.RecordRef{Cid: parentRef.Cid},
	}
	childData := `{"name": "child"}`
	childRef, err := store.Push(testCtx, childMeta, io.NopCloser(strings.NewReader(childData)))
	assert.NoError(t, err, "Child push should succeed")

	// Pull child and verify parent reference
	pulledChildMeta, childReader, err := store.Pull(testCtx, childRef)
	assert.NoError(t, err, "Child pull should succeed")
	defer childReader.Close()

	assert.NotNil(t, pulledChildMeta.Parent, "Child should have parent reference")
	assert.Equal(t, parentRef.Cid, pulledChildMeta.Parent.Cid, "Parent CID should match")
}

// testPushComplexDAG tests creating a multi-level DAG with both links and parents.
func TestPushComplexDAG(t *testing.T) {
	store := loadLocalStore(t)

	// Level 3: Leaf nodes (data)
	leaf1Meta := &corev1.RecordMeta{
		Cid:       "bafyleaf1",
		Type:      "test.data.v1",
		CreatedAt: "2025-12-03T10:00:00Z",
		Size:      20,
	}
	leaf1Ref, err := store.Push(testCtx, leaf1Meta, io.NopCloser(strings.NewReader(`{"value": 1}`)))
	assert.NoError(t, err)

	leaf2Meta := &corev1.RecordMeta{
		Cid:       "bafyleaf2",
		Type:      "test.data.v1",
		CreatedAt: "2025-12-03T10:00:01Z",
		Size:      20,
	}
	leaf2Ref, err := store.Push(testCtx, leaf2Meta, io.NopCloser(strings.NewReader(`{"value": 2}`)))
	assert.NoError(t, err)

	// Level 2: Middle nodes (link to leaves)
	mid1Meta := &corev1.RecordMeta{
		Cid:       "bafymid1",
		Type:      "test.intermediate.v1",
		CreatedAt: "2025-12-03T10:01:00Z",
		Size:      50,
		Links: []*corev1.RecordRef{
			{Cid: leaf1Ref.Cid},
		},
	}
	mid1Ref, err := store.Push(testCtx, mid1Meta, io.NopCloser(strings.NewReader(`{"layer": "middle1"}`)))
	assert.NoError(t, err)

	mid2Meta := &corev1.RecordMeta{
		Cid:       "bafymid2",
		Type:      "test.intermediate.v1",
		CreatedAt: "2025-12-03T10:01:01Z",
		Size:      50,
		Links: []*corev1.RecordRef{
			{Cid: leaf2Ref.Cid},
		},
	}
	mid2Ref, err := store.Push(testCtx, mid2Meta, io.NopCloser(strings.NewReader(`{"layer": "middle2"}`)))
	assert.NoError(t, err)

	// Level 1: Root node (links to middle nodes)
	rootMeta := &corev1.RecordMeta{
		Cid:       "bafyroot",
		Type:      "test.root.v1",
		CreatedAt: "2025-12-03T10:02:00Z",
		Size:      100,
		Links: []*corev1.RecordRef{
			{Cid: mid1Ref.Cid},
			{Cid: mid2Ref.Cid},
		},
		Annotations: map[string]string{
			"dag.level": "root",
			"dag.depth": "3",
		},
	}
	rootRef, err := store.Push(testCtx, rootMeta, io.NopCloser(strings.NewReader(`{"layer": "root"}`)))
	assert.NoError(t, err)

	// Verify DAG structure by traversing from root
	rootPulled, _, err := store.Pull(testCtx, rootRef)
	assert.NoError(t, err)
	assert.Len(t, rootPulled.Links, 2, "Root should have 2 children")
	assert.Equal(t, "3", rootPulled.Annotations["dag.depth"])

	// Verify middle layer
	mid1Pulled, _, err := store.Pull(testCtx, &corev1.RecordRef{Cid: mid1Ref.Cid})
	assert.NoError(t, err)
	assert.Len(t, mid1Pulled.Links, 1, "Mid1 should have 1 child")
	assert.Equal(t, leaf1Ref.Cid, mid1Pulled.Links[0].Cid)

	mid2Pulled, _, err := store.Pull(testCtx, &corev1.RecordRef{Cid: mid2Ref.Cid})
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

	// Push a record
	meta := &corev1.RecordMeta{
		Cid:       "bafylookup",
		Type:      "test.lookup.v1",
		CreatedAt: "2025-12-03T10:00:00Z",
		Size:      100,
		Annotations: map[string]string{
			"test": "lookup",
		},
	}
	data := `{"test": "data"}`
	ref, err := store.Push(testCtx, meta, io.NopCloser(strings.NewReader(data)))
	assert.NoError(t, err)

	// Lookup metadata
	lookedUpMeta, err := store.Lookup(testCtx, ref)
	assert.NoError(t, err, "Lookup should succeed")
	assert.NotNil(t, lookedUpMeta, "Looked up metadata should not be nil")
	assert.Equal(t, meta.Type, lookedUpMeta.Type)
	assert.Equal(t, meta.CreatedAt, lookedUpMeta.CreatedAt)
	assert.Equal(t, meta.Annotations, lookedUpMeta.Annotations)
}

// testDeleteRecord tests deleting a record from the store.
func TestDeleteRecord(t *testing.T) {
	store := loadLocalStore(t)

	// Push a record
	meta := &corev1.RecordMeta{
		Cid:       "bafydelete",
		Type:      "test.delete.v1",
		CreatedAt: "2025-12-03T10:00:00Z",
		Size:      50,
	}
	data := `{"test": "delete"}`
	ref, err := store.Push(testCtx, meta, io.NopCloser(strings.NewReader(data)))
	assert.NoError(t, err)

	// Verify record exists
	_, err = store.Lookup(testCtx, ref)
	assert.NoError(t, err, "Record should exist before deletion")

	// Delete record
	err = store.Delete(testCtx, ref)
	assert.NoError(t, err, "Delete should succeed")

	// Verify record no longer exists
	_, err = store.Lookup(testCtx, ref)
	assert.Error(t, err, "Lookup should fail after deletion")
}

// testDeleteNonExistentRecord tests deleting a record that doesn't exist.
func TestDeleteNonExistentRecord(t *testing.T) {
	store := loadLocalStore(t)

	// Try to delete non-existent record
	ref := &corev1.RecordRef{Cid: "bafynonexistent"}
	err := store.Delete(testCtx, ref)
	assert.Error(t, err, "Delete should fail for non-existent record")
}

func loadLocalStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create local
	store, err := New(ociconfig.Config{LocalDir: "./testdata/oci_local_store"})
	assert.NoErrorf(t, err, "failed to create local store")

	return store
}
