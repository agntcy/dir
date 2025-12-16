// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package oci

import (
	"context"
	"fmt"
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
	// Blob/Layer tests - basic content storage
	t.Run("Blob/PushPullSimpleBlob", TestPushPullSimpleBlob)
	t.Run("Blob/PushMultipleBlobs", TestPushMultipleBlobs)
	t.Run("Blob/LookupBlob", TestLookupBlob)

	// Manifest tests - structured objects with metadata and traversal
	t.Run("Manifest/PushPullManifest", TestPushPullManifest)
	t.Run("Manifest/LookupManifestMetadata", TestLookupManifestMetadata)
	t.Run("Manifest/WalkManifestLayers", TestWalkManifestLayers)
}

// TestPushPullSimpleBlob tests pushing and pulling a basic blob (layer).
// Blobs are raw content without OCI structure - they only store the data.
func TestPushPullSimpleBlob(t *testing.T) {
	store := loadLocalStore(t)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"
	data := `{"name": "test-agent", "version": "1.0.0"}`
	dataReader := io.NopCloser(strings.NewReader(data))

	// Push blob
	ref, err := store.Push(testCtx, mediaType, dataReader)
	assert.NoError(t, err, "Push should succeed")
	assert.NotNil(t, ref, "ObjectRef should not be nil")
	assert.NotEmpty(t, ref.Cid, "CID should not be empty")

	// Pull blob back
	pulledMeta, pulledDataReader, err := store.Pull(testCtx, ref)
	assert.NoError(t, err, "Pull should succeed")
	assert.NotNil(t, pulledMeta, "Pulled metadata should not be nil")
	assert.NotNil(t, pulledDataReader, "Pulled data reader should not be nil")
	defer pulledDataReader.Close()

	// Verify metadata (blobs have basic metadata only)
	assert.Equal(t, ref.Cid, pulledMeta.Cid, "CID should match")
	assert.Equal(t, mediaType, pulledMeta.MediaType, "MediaType should match")
	assert.Greater(t, pulledMeta.Size, uint64(0), "Size should be greater than 0")
	assert.Empty(t, pulledMeta.ArtifactType, "Blobs should not have artifact type")
	assert.Empty(t, pulledMeta.Annotations, "Blobs should not have annotations")

	// Verify data content
	pulledData, err := io.ReadAll(pulledDataReader)
	assert.NoError(t, err, "Reading pulled data should succeed")
	assert.Equal(t, data, string(pulledData), "Data content should match")
}

// TestPushMultipleBlobs tests pushing multiple blobs and verifying they can all be pulled.
// Each blob is stored independently as raw content.
func TestPushMultipleBlobs(t *testing.T) {
	store := loadLocalStore(t)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"

	// Create multiple blobs
	blob1Data := `{"data": "blob1"}`
	blob1Ref, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(blob1Data)))
	assert.NoError(t, err, "Blob1 push should succeed")

	blob2Data := `{"data": "blob2"}`
	blob2Ref, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(blob2Data)))
	assert.NoError(t, err, "Blob2 push should succeed")

	blob3Data := `{"name": "blob3"}`
	blob3Ref, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(blob3Data)))
	assert.NoError(t, err, "Blob3 push should succeed")

	// Verify all blobs are independently accessible
	_, blob1Reader, err := store.Pull(testCtx, blob1Ref)
	assert.NoError(t, err, "Blob1 should be pullable")
	blob1Reader.Close()

	_, blob2Reader, err := store.Pull(testCtx, blob2Ref)
	assert.NoError(t, err, "Blob2 should be pullable")
	blob2Reader.Close()

	_, blob3Reader, err := store.Pull(testCtx, blob3Ref)
	assert.NoError(t, err, "Blob3 should be pullable")
	blob3Reader.Close()
}

// TestLookupBlob tests the Lookup operation for blobs - retrieving basic metadata only.
// Blobs return minimal metadata without annotations or artifact type.
func TestLookupBlob(t *testing.T) {
	store := loadLocalStore(t)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"
	data := `{"test": "blob-data"}`

	// Push a blob
	ref, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(data)))
	assert.NoError(t, err)

	// Lookup metadata (without pulling data)
	lookedUpMeta, err := store.Lookup(testCtx, ref)
	assert.NoError(t, err, "Lookup should succeed")
	assert.NotNil(t, lookedUpMeta, "Looked up metadata should not be nil")
	assert.Equal(t, ref.Cid, lookedUpMeta.Cid, "CID should match")
	assert.Equal(t, mediaType, lookedUpMeta.MediaType, "MediaType should match")
	assert.Greater(t, lookedUpMeta.Size, uint64(0), "Size should be greater than 0")

	// Blobs should not have rich metadata
	assert.Empty(t, lookedUpMeta.ArtifactType, "Blobs should not have artifact type")
	assert.Empty(t, lookedUpMeta.Annotations, "Blobs should not have annotations")
}

// TestPushPullManifest tests pushing and pulling an OCI manifest.
// Manifests are structured objects that can contain annotations and reference other objects.
func TestPushPullManifest(t *testing.T) {
	store := loadLocalStore(t)

	// First push some blobs that the manifest will reference
	layer1Data := `{"content": "layer1"}`
	layer1Ref, err := store.Push(testCtx, "application/vnd.oci.image.layer.v1.tar+gzip",
		io.NopCloser(strings.NewReader(layer1Data)))
	assert.NoError(t, err)

	// Create a manifest that references the layer
	manifestData := `{
		"schemaVersion": 2,
		"mediaType": "application/vnd.oci.image.manifest.v1+json",
		"artifactType": "application/vnd.agntcy.dir.record.v1+json",
		"config": {
			"mediaType": "application/vnd.oci.empty.v1+json",
			"digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			"size": 2
		},
		"layers": [
			{
				"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
				"digest": "` + layer1Ref.Cid + `",
				"size": ` + fmt.Sprintf("%d", len(layer1Data)) + `
			}
		],
		"annotations": {
			"org.opencontainers.image.created": "2025-12-08T10:00:00Z",
			"custom.annotation": "test-value"
		}
	}`

	// Push manifest
	manifestRef, err := store.Push(testCtx, "application/vnd.oci.image.manifest.v1+json",
		io.NopCloser(strings.NewReader(manifestData)))
	assert.NoError(t, err, "Manifest push should succeed")
	assert.NotNil(t, manifestRef, "Manifest ref should not be nil")

	// Pull manifest back
	pulledMeta, pulledReader, err := store.Pull(testCtx, manifestRef)
	assert.NoError(t, err, "Manifest pull should succeed")
	assert.NotNil(t, pulledMeta, "Pulled metadata should not be nil")
	assert.NotNil(t, pulledReader, "Pulled reader should not be nil")
	defer pulledReader.Close()

	// Verify manifest metadata (manifests have rich metadata)
	assert.Equal(t, manifestRef.Cid, pulledMeta.Cid, "CID should match")
	assert.Equal(t, "application/vnd.oci.image.manifest.v1+json", pulledMeta.MediaType, "MediaType should match")
	assert.Equal(t, "application/vnd.agntcy.dir.record.v1+json", pulledMeta.ArtifactType, "ArtifactType should match")
	assert.NotEmpty(t, pulledMeta.Annotations, "Manifest should have annotations")
	assert.Equal(t, "test-value", pulledMeta.Annotations["custom.annotation"], "Custom annotation should match")

	// Verify manifest content
	pulledData, err := io.ReadAll(pulledReader)
	assert.NoError(t, err, "Reading manifest data should succeed")
	assert.Contains(t, string(pulledData), "schemaVersion", "Manifest should contain schemaVersion")
}

// TestLookupManifestMetadata tests that manifests return rich metadata from Lookup.
// Unlike blobs, manifests expose annotations and artifact type.
func TestLookupManifestMetadata(t *testing.T) {
	store := loadLocalStore(t)

	// Create a simple manifest with annotations
	manifestData := `{
		"schemaVersion": 2,
		"mediaType": "application/vnd.oci.image.manifest.v1+json",
		"artifactType": "application/vnd.agntcy.test.artifact.v1+json",
		"config": {
			"mediaType": "application/vnd.oci.empty.v1+json",
			"digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			"size": 2
		},
		"layers": [],
		"annotations": {
			"test.key1": "value1",
			"test.key2": "value2",
			"test.timestamp": "2025-12-08T15:00:00Z"
		}
	}`

	// Push manifest
	ref, err := store.Push(testCtx, "application/vnd.oci.image.manifest.v1+json",
		io.NopCloser(strings.NewReader(manifestData)))
	assert.NoError(t, err)

	// Lookup metadata without pulling data
	meta, err := store.Lookup(testCtx, ref)
	assert.NoError(t, err, "Lookup should succeed")
	assert.NotNil(t, meta, "Metadata should not be nil")

	// Verify rich metadata is available
	assert.Equal(t, ref.Cid, meta.Cid, "CID should match")
	assert.Equal(t, "application/vnd.oci.image.manifest.v1+json", meta.MediaType, "MediaType should match")
	assert.Equal(t, "application/vnd.agntcy.test.artifact.v1+json", meta.ArtifactType, "ArtifactType should be populated")
	assert.Len(t, meta.Annotations, 3, "Should have 3 annotations")
	assert.Equal(t, "value1", meta.Annotations["test.key1"])
	assert.Equal(t, "value2", meta.Annotations["test.key2"])
	assert.Equal(t, "2025-12-08T15:00:00Z", meta.Annotations["test.timestamp"])
}

// TestWalkManifestLayers tests the Walk operation on manifests.
// Walk allows traversing the layers/objects referenced by a manifest.
// Note: Walk is only meaningful for manifests, not for blobs.
func TestWalkManifestLayers(t *testing.T) {
	t.Skip("Walk implementation is not yet complete - test will be implemented when Walk is functional")

	store := loadLocalStore(t)

	// Create multiple layers
	layer1Ref, err := store.Push(testCtx, "application/vnd.oci.image.layer.v1.tar+gzip",
		io.NopCloser(strings.NewReader(`{"data": "layer1"}`)))
	assert.NoError(t, err)

	layer2Ref, err := store.Push(testCtx, "application/vnd.oci.image.layer.v1.tar+gzip",
		io.NopCloser(strings.NewReader(`{"data": "layer2"}`)))
	assert.NoError(t, err)

	// Create manifest referencing both layers
	manifestData := fmt.Sprintf(`{
		"schemaVersion": 2,
		"mediaType": "application/vnd.oci.image.manifest.v1+json",
		"config": {
			"mediaType": "application/vnd.oci.empty.v1+json",
			"digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
			"size": 2
		},
		"layers": [
			{"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "%s", "size": 20},
			{"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "digest": "%s", "size": 20}
		]
	}`, layer1Ref.Cid, layer2Ref.Cid)

	manifestRef, err := store.Push(testCtx, "application/vnd.oci.image.manifest.v1+json",
		io.NopCloser(strings.NewReader(manifestData)))
	assert.NoError(t, err)

	// Walk through the manifest's layers
	visited := make([]string, 0)
	err = store.Walk(testCtx, manifestRef, func(meta *storev1.ObjectMeta) error {
		visited = append(visited, meta.Cid)
		return nil
	})
	assert.NoError(t, err, "Walk should succeed")

	// Verify we visited all layers
	assert.Contains(t, visited, layer1Ref.Cid, "Should have visited layer1")
	assert.Contains(t, visited, layer2Ref.Cid, "Should have visited layer2")
}

func loadLocalStore(t *testing.T) types.StoreAPI {
	t.Helper()

	// create local
	store, err := New(ociconfig.Config{LocalDir: "./testdata/oci_local_store"})
	assert.NoErrorf(t, err, "failed to create local store")

	return store
	// return loadRemoteStore(t)
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

// BenchmarkPushSimpleBlobs benchmarks pushing simple blob objects.
func BenchmarkPushSimpleBlobs(b *testing.B) {
	store := loadBenchStore(b)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"
	sampleData := `{"name": "benchmark-object", "version": "1.0.0", "description": "Sample benchmark data"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(sampleData)))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}
	}
}

// BenchmarkPushMultipleBlobs benchmarks pushing multiple blob objects sequentially.
func BenchmarkPushMultipleBlobs(b *testing.B) {
	store := loadBenchStore(b)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"

	// Create leaf objects once for reuse
	leafData := `{"data": "leaf-content"}`
	leafRef1, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(leafData)))
	if err != nil {
		b.Fatalf("Failed to create leaf1: %v", err)
	}
	leafRef2, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(leafData)))
	if err != nil {
		b.Fatalf("Failed to create leaf2: %v", err)
	}

	// Prevent unused variable errors
	_ = leafRef1
	_ = leafRef2

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parentData := `{"name": "parent-with-links"}`
		_, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(parentData)))
		if err != nil {
			b.Fatalf("Push failed: %v", err)
		}
	}
}

// BenchmarkPushPullCycleBlob benchmarks a complete push-pull cycle for blobs.
func BenchmarkPushPullCycleBlob(b *testing.B) {
	store := loadBenchStore(b)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"
	sampleData := `{"test": "benchmark-cycle-data"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Push
		ref, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(sampleData)))
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

// BenchmarkLookupBlob benchmarks metadata lookup operations for blobs.
func BenchmarkLookupBlob(b *testing.B) {
	store := loadBenchStore(b)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"
	sampleData := `{"test": "lookup-data"}`

	// Push objects first
	refs := make([]*storev1.ObjectRef, 100)
	for i := 0; i < 100; i++ {
		ref, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(sampleData)))
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

// BenchmarkPushLargeBlobData benchmarks pushing blob objects with larger data payloads.
func BenchmarkPushLargeBlobData(b *testing.B) {
	store := loadBenchStore(b)

	mediaType := "application/vnd.oci.image.layer.v1.tar+gzip"

	// Create 100KB sample data
	largeData := strings.Repeat(`{"key": "value", "data": "sample content"}`, 2500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.Push(testCtx, mediaType, io.NopCloser(strings.NewReader(largeData)))
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
