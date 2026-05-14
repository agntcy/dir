// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package controller

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	corev1 "github.com/agntcy/dir/api/core/v1"
	storev2 "github.com/agntcy/dir/api/store/v2"
	"github.com/agntcy/dir/server/store/oci"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/store/packaging"
	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/opencontainers/go-digest"
	specs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestObjStoreBlob verifies blob (non-manifest) push lifecycle: push, has, get,
// raw get, lookup, list referrers (empty since blobs are not manifests),
// and delete.
func TestObjStoreBlob(t *testing.T) {
	store := getCtrl(t)
	ctx := t.Context()

	// Push a blob
	blobContent := []byte("hello-world")
	blobDigest := digest.FromBytes(blobContent).String()

	desc, err := store.Put(ctx, &storev2.Object{
		Size: uint64(len(blobContent)),
		Data: blobContent,
	})
	require.NoError(t, err)
	assert.Equal(t, blobDigest, desc.GetDigest())
	assert.Equal(t, uint64(len(blobContent)), desc.GetSize())

	// Has the blob
	has, err := store.Has(ctx, &storev2.ObjectRef{Cid: blobDigest})
	require.NoError(t, err)
	assert.True(t, has.GetValue())

	// Get the blob (no manifest -> falls back to raw blob fetch)
	got, err := store.Get(ctx, &storev2.ObjectRef{Cid: blobDigest})
	require.NoError(t, err)
	assert.Equal(t, blobContent, got.GetData())
	assert.Equal(t, "application/octet-stream", got.GetMediaType())

	// GetRaw returns the same bytes
	gotRaw, err := store.GetRaw(ctx, &storev2.ObjectRef{Cid: blobDigest})
	require.NoError(t, err)
	assert.Equal(t, blobContent, gotRaw.GetData())

	// Lookup returns basic descriptor info
	meta, err := store.Lookup(ctx, &storev2.ObjectRef{Cid: blobDigest})
	require.NoError(t, err)
	assert.Equal(t, blobDigest, meta.GetDigest())
	assert.Equal(t, uint64(len(blobContent)), meta.GetSize())

	// ListReferrers is not supported for blobs
	referrers, err := store.ListReferrers(ctx, &storev2.ListReferrersRequest{Subject: &storev2.ObjectRef{Cid: blobDigest}})
	require.NoError(t, err)
	assert.Empty(t, referrers.GetReferrers())

	// Delete the blob
	_, err = store.Delete(ctx, &storev2.ObjectRef{Cid: blobDigest})
	require.NoError(t, err)

	// Has should now report false
	has, err = store.Has(ctx, &storev2.ObjectRef{Cid: blobDigest})
	require.NoError(t, err)
	assert.False(t, has.GetValue())
}

// TestObjStoreManifest verifies pushing a plain OCI manifest (with config+layer
// blobs) plus pushing a referrer manifest that points at it via Subject.
func TestObjStoreManifest(t *testing.T) {
	store := getCtrl(t)
	ctx := t.Context()

	// Push config blob
	configContent := []byte(`{"architecture":"amd64","os":"linux"}`)
	configDesc, err := store.Put(ctx, &storev2.Object{
		MediaType: "application/vnd.oci.image.config.v1+json",
		Size:      uint64(len(configContent)),
		Data:      configContent,
	})
	require.NoError(t, err)

	// Push layer blob
	layerContent := []byte("layer-content")
	layerDesc, err := store.Put(ctx, &storev2.Object{
		MediaType: "application/vnd.oci.image.layer.v1.tar",
		Size:      uint64(len(layerContent)),
		Data:      layerContent,
	})
	require.NoError(t, err)

	// Build an OCI image manifest with the config + a single layer
	manifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: "application/vnd.test.artifact",
		Config: ocispec.Descriptor{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    digest.Digest(configDesc.GetDigest()),
			Size:      int64(len(configContent)),
		},
		Layers: []ocispec.Descriptor{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				Digest:    digest.Digest(layerDesc.GetDigest()),
				Size:      int64(len(layerContent)),
				Annotations: map[string]string{
					"org.test.layer": "1",
				},
			},
		},
		Annotations: map[string]string{
			"org.test.manifest": "true",
		},
	}

	manifestData, err := json.Marshal(manifest)
	require.NoError(t, err)

	// Push the manifest itself
	manifestDesc, err := store.Put(ctx, &storev2.Object{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      uint64(len(manifestData)),
		Data:      manifestData,
	})
	require.NoError(t, err)

	manifestDigest := manifestDesc.GetDigest()
	assert.Equal(t, manifestDigest, manifestDesc.GetDigest())

	// Has the manifest
	has, err := store.Has(ctx, &storev2.ObjectRef{Cid: manifestDigest})
	require.NoError(t, err)
	assert.True(t, has.GetValue())

	// Get the manifest. The artifact type is not registered with any packer, so
	// Get falls back to returning the raw manifest bytes.
	gotObj, err := store.Get(ctx, &storev2.ObjectRef{Cid: manifestDigest})
	require.NoError(t, err)
	assert.Equal(t, manifestData, gotObj.GetData())

	// GetRaw returns the same manifest bytes
	gotRaw, err := store.GetRaw(ctx, &storev2.ObjectRef{Cid: manifestDigest})
	require.NoError(t, err)
	assert.Equal(t, manifestData, gotRaw.GetData())

	// Lookup returns manifest metadata
	meta, err := store.Lookup(ctx, &storev2.ObjectRef{Cid: manifestDigest})
	require.NoError(t, err)
	assert.Equal(t, manifestDigest, meta.GetDigest())
	assert.Equal(t, uint64(len(manifestData)), meta.GetSize())
	assert.Equal(t, ocispec.MediaTypeImageManifest, meta.GetMediaType())
	assert.Equal(t, "application/vnd.test.artifact", meta.GetArtifactType())
	assert.Equal(t, "true", meta.GetAnnotations()["org.test.manifest"])

	// No referrers yet
	refs, err := store.ListReferrers(ctx, &storev2.ListReferrersRequest{Subject: &storev2.ObjectRef{Cid: manifestDigest}})
	require.NoError(t, err)
	assert.Empty(t, refs.GetReferrers())

	// Build a referrer manifest with Subject pointing to the manifest above
	referrerManifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: "application/vnd.test.referrer",
		Config:       ocispec.DescriptorEmptyJSON,
		Layers:       []ocispec.Descriptor{ocispec.DescriptorEmptyJSON},
		Subject: &ocispec.Descriptor{
			MediaType:    ocispec.MediaTypeImageManifest,
			ArtifactType: manifest.ArtifactType,
			Digest:       digest.Digest(manifestDigest),
			Size:         int64(len(manifestData)),
		},
		Annotations: map[string]string{
			"org.test.referrer": "yes",
		},
	}

	referrerData, err := json.Marshal(referrerManifest)
	require.NoError(t, err)

	referrerDesc, err := store.Put(ctx, &storev2.Object{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      uint64(len(referrerData)),
		Data:      referrerData,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, referrerDesc.GetDigest())

	// ListReferrers should now include the new referrer
	refs, err = store.ListReferrers(ctx, &storev2.ListReferrersRequest{Subject: &storev2.ObjectRef{Cid: manifestDigest}})
	require.NoError(t, err)
	require.Len(t, refs.GetReferrers(), 1)
	assert.Equal(t, referrerDesc.GetDigest(), refs.GetReferrers()[0].GetDigest())

	// TODO: The referrer manifest's ArtifactType is not available for all OCI registries (e.g. in-memory),
	// so this assertion is currently unreliable. Clients can use Lookup on the referrer descriptor to get the artifact type instead.
	// For OCI stores that do not index over artifact type, filteration also does not work.
	// assert.Equal(t, "application/vnd.test.referrer", refs.GetReferrers()[0].GetArtifactType())

	// Delete referrer manifest
	_, err = store.Delete(ctx, &storev2.ObjectRef{Cid: referrerDesc.GetDigest()})
	require.NoError(t, err)

	// ListReferrers should now be empty again
	refs, err = store.ListReferrers(ctx, &storev2.ListReferrersRequest{Subject: &storev2.ObjectRef{Cid: manifestDigest}})
	require.NoError(t, err)
	assert.Empty(t, refs.GetReferrers())

	// Remove the original manifest
	_, err = store.Delete(ctx, &storev2.ObjectRef{Cid: manifestDigest})
	require.NoError(t, err)

	has, err = store.Has(ctx, &storev2.ObjectRef{Cid: manifestDigest})
	require.NoError(t, err)
	assert.False(t, has.GetValue())
}

// TestObjStoreRecord verifies that pushing an object with the record media type
// runs through the record packer: the resulting descriptor is an OCI manifest
// with ArtifactType=RecordMediaType, Get returns the unpacked record bytes,
// GetRaw returns the raw manifest bytes, and Lookup exposes record annotations.
func TestObjStoreRecord(t *testing.T) {
	store := getCtrl(t)
	ctx := t.Context()

	// Build a sample record and marshal it via the canonical record marshaler
	record := corev1.New(&typesv1alpha1.Record{
		Name:          "test-agent",
		Version:       "v1.0.0",
		SchemaVersion: "0.7.0",
		Description:   "A test agent",
		CreatedAt:     "2023-01-01T00:00:00Z",
		Annotations: map[string]string{
			"team": "controller-test",
		},
	})
	recordBytes, err := record.Marshal()
	require.NoError(t, err)

	// Push the record object. The record packer will produce an OCI manifest
	// referencing the record bytes as a layer.
	desc, err := store.Put(ctx, &storev2.Object{
		MediaType: packaging.RecordMediaType,
		Size:      uint64(len(recordBytes)),
		Data:      recordBytes,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, desc.GetDigest())
	assert.Equal(t, ocispec.MediaTypeImageManifest, desc.GetMediaType())
	assert.Equal(t, packaging.RecordMediaType, desc.GetArtifactType())

	// Has the manifest
	has, err := store.Has(ctx, &storev2.ObjectRef{Cid: desc.GetDigest()})
	require.NoError(t, err)
	assert.True(t, has.GetValue())

	// Get returns the unpacked record bytes (record packer Unpack)
	got, err := store.Get(ctx, &storev2.ObjectRef{Cid: desc.GetDigest()})
	require.NoError(t, err)
	assert.Equal(t, packaging.RecordMediaType, got.GetMediaType())
	assert.Equal(t, recordBytes, got.GetData())

	// GetRaw returns the raw manifest bytes (not the record)
	gotRaw, err := store.GetRaw(ctx, &storev2.ObjectRef{Cid: desc.GetDigest()})
	require.NoError(t, err)
	assert.Equal(t, ocispec.MediaTypeImageManifest, gotRaw.GetMediaType())

	var rawManifest ocispec.Manifest
	require.NoError(t, json.Unmarshal(gotRaw.GetData(), &rawManifest))
	assert.Equal(t, packaging.RecordMediaType, rawManifest.ArtifactType)
	require.Len(t, rawManifest.Layers, 1)

	// Lookup returns manifest metadata with the record annotations
	meta, err := store.Lookup(ctx, &storev2.ObjectRef{Cid: desc.GetDigest()})
	require.NoError(t, err)
	assert.Equal(t, packaging.RecordMediaType, meta.GetArtifactType())

	// No referrers
	refs, err := store.ListReferrers(ctx, &storev2.ListReferrersRequest{Subject: &storev2.ObjectRef{Cid: desc.GetDigest()}})
	require.NoError(t, err)
	assert.Empty(t, refs.GetReferrers())

	// Cleanup
	_, err = store.Delete(ctx, &storev2.ObjectRef{Cid: desc.GetDigest()})
	require.NoError(t, err)

	for _, layer := range rawManifest.Layers {
		_, _ = store.Delete(ctx, &storev2.ObjectRef{Cid: layer.Digest.String()})
	}
}

// TestObjStoreManifestRecordReferrer verifies that a record can act as a
// referrer for another manifest: ListReferrers on the subject returns the
// record descriptor, and Get on that descriptor returns the original Record
// object (unpacked by the record packer).
func TestObjStoreManifestRecordReferrer(t *testing.T) {
	store := getCtrl(t)
	ctx := t.Context()

	// Push a base manifest that will act as the subject. Use a minimal OCI
	// image manifest with empty config/layers.
	baseManifest := ocispec.Manifest{
		Versioned:    specs.Versioned{SchemaVersion: 2},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: "application/vnd.test.base",
		Config:       ocispec.DescriptorEmptyJSON,
		Layers:       []ocispec.Descriptor{ocispec.DescriptorEmptyJSON},
		Annotations: map[string]string{
			"org.test.base": "true",
		},
	}
	baseData, err := json.Marshal(baseManifest)
	require.NoError(t, err)

	baseDesc, err := store.Put(ctx, &storev2.Object{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      uint64(len(baseData)),
		Data:      baseData,
	})
	require.NoError(t, err)

	// Build a record to attach as a referrer
	record := corev1.New(&typesv1alpha1.Record{
		Name:              "referrer-agent",
		Version:           "v0.1.0",
		SchemaVersion:     "0.7.0",
		Description:       "A record acting as a referrer",
		PreviousRecordCid: new(baseDesc.GetDigest()),
	})
	recordBytes, err := record.Marshal()
	require.NoError(t, err)

	// Push the record bytes as a raw JSON blob; this becomes the manifest layer.
	recordObject := &storev2.Object{
		MediaType: packaging.RecordMediaType,
		Size:      uint64(len(recordBytes)),
		Data:      recordBytes,
	}
	recordDesc, err := store.Put(ctx, recordObject)
	require.NoError(t, err)

	// ListReferrers on the base manifest should return the record manifest
	refs, err := store.ListReferrers(ctx, &storev2.ListReferrersRequest{Subject: &storev2.ObjectRef{Cid: baseDesc.GetDigest()}})
	require.NoError(t, err)
	require.Len(t, refs.GetReferrers(), 1)
	referrer := refs.GetReferrers()[0]
	assert.Equal(t, recordDesc.GetDigest(), referrer.GetDigest())
	assert.Equal(t, recordDesc.GetMediaType(), referrer.GetMediaType())
	assert.Equal(t, recordDesc.GetArtifactType(), referrer.GetArtifactType())
	assert.Equal(t, recordDesc.GetAnnotations(), referrer.GetAnnotations())

	// Get on the referrer should unpack and return the original Record object
	got, err := store.Get(ctx, &storev2.ObjectRef{Cid: referrer.GetDigest()})
	require.NoError(t, err)
	assert.Equal(t, recordObject.GetMediaType(), got.GetMediaType())
	assert.Equal(t, recordObject.GetData(), got.GetData())

	// Validate that the unpacked bytes are a parseable record matching the original
	gotRecord, err := corev1.UnmarshalRecord(got.GetData())
	require.NoError(t, err)
	assert.Equal(t, record.GetCid(), gotRecord.GetCid())

	// Cleanup
	_, err = store.Delete(ctx, &storev2.ObjectRef{Cid: recordDesc.GetDigest()})
	require.NoError(t, err)
	_, err = store.Delete(ctx, &storev2.ObjectRef{Cid: baseDesc.GetDigest()})
	require.NoError(t, err)
}

// getCtrl creates an ObjectStoreServer backed by a fresh OCI repository
// via in-memory registry. Each test gets its own repository name to avoid
// cross test interference.
func getCtrl(t *testing.T) storev2.ObjectStoreServer {
	t.Helper()

	// Start registry
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "localhost:0")
	require.NoError(t, err)

	// Start a local in-memory registry on the listener
	s := &http.Server{
		ReadHeaderTimeout: 5 * time.Second, // prevent slowloris, quiet linter
		Handler:           registry.New(registry.WithReferrersSupport(true)),
	}

	t.Cleanup(func() {
		s.Close()
		listener.Close()
	})

	go func() {
		_ = s.Serve(listener)
	}()

	// Create repository to use
	//nolint:forcetypeassert
	repo, err := oci.NewORASRepository(ociconfig.Config{
		RegistryAddress: fmt.Sprintf("localhost:%d", listener.Addr().(*net.TCPAddr).Port),
		RepositoryName:  fmt.Sprintf("controller-test-%d", time.Now().UnixNano()),
		AuthConfig: ociconfig.AuthConfig{
			Insecure: true,
		},
	})
	require.NoError(t, err)

	store := NewObjStore(repo)

	// Push the empty JSON blob ({}) so the referrer manifest's Config/Layers
	// (which reuse ocispec.DescriptorEmptyJSON) point at an existing blob.
	// Some registries (e.g. zot) reject manifests whose referenced blobs are
	// missing with HTTP 400 "manifest invalid".
	_, err = store.Put(t.Context(), &storev2.Object{
		MediaType: ocispec.DescriptorEmptyJSON.MediaType,
		Size:      uint64(len(ocispec.DescriptorEmptyJSON.Data)),
		Data:      ocispec.DescriptorEmptyJSON.Data,
	})
	require.NoError(t, err)

	return store
}
