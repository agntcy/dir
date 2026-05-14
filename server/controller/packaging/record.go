// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package packaging

import (
	"context"
	"io"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev2 "github.com/agntcy/dir/api/store/v2"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

const RecordMediaType = "application/vnd.agntcy.dir.v1+json"

type record struct{}

func init() {
	RegisterPacker(RecordMediaType, &record{})
}

// Pack implements [Packer].
func (r *record) Pack(ctx context.Context, repo *remote.Repository, obj *storev2.Object) (*ocispec.Manifest, error) {
	// Convert object to a record
	record, err := corev1.UnmarshalRecord(obj.GetData())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to unmarshal object data into record: %v", err)
	}

	// Marshal the record using canonical JSON marshaling first
	// This ensures consistent bytes for both CID calculation and storage
	recordBytes, err := record.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	// Step 1: Use oras.PushBytes to push the record data and get Layer Descriptor
	layerDesc, err := oras.PushBytes(ctx, repo, "application/json", recordBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push record bytes: %v", err)
	}

	// Step 3: Construct manifest annotations and add CID to annotations
	manifestAnnotations := extractManifestAnnotations(record)

	return &ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: int(oras.PackManifestVersion1_1),
		},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: RecordMediaType,
		Config:       ocispec.DescriptorEmptyJSON,
		Layers: []ocispec.Descriptor{
			layerDesc,
		},
		Subject:     nil,
		Annotations: manifestAnnotations,
	}, nil
}

// Unpack implements [Packer].
func (r *record) Unpack(ctx context.Context, repo *remote.Repository, manifest *ocispec.Manifest) (*storev2.Object, error) {
	// Validate manifest has exactly one layer
	if len(manifest.Layers) != 1 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid manifest: expected exactly 1 layer, got %d", len(manifest.Layers))
	}

	// Pull the layer content as bytes
	layerReader, err := repo.Fetch(ctx, manifest.Layers[0])
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch layer content: %v", err)
	}
	defer layerReader.Close()

	layerBytes, err := io.ReadAll(layerReader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read layer content: %v", err)
	}

	// Validate record
	_, err = corev1.UnmarshalRecord(layerBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal layer bytes into record: %v", err)
	}

	return &storev2.Object{
		MediaType: RecordMediaType,
		Size:      uint64(len(layerBytes)),
		Data:      layerBytes,
	}, nil
}

// extractManifestAnnotations extracts manifest annotations from record using adapter pattern.
//
//nolint:cyclop // Function handles multiple annotation types with justified complexity
func extractManifestAnnotations(record *corev1.Record) map[string]string {
	annotations := make(map[string]string)

	// Always set the type
	annotations[manifestDirObjectTypeKey] = "record"

	// Use adapter pattern to get version-agnostic access to record data
	adapter := adapters.NewRecordAdapter(record)

	recordData, err := adapter.GetRecordData()
	if err != nil {
		// Return minimal annotations if no valid data
		return annotations
	}

	// Add version details
	annotations[ManifestKeyOASFVersion] = record.GetSchemaVersion()

	// Core identity fields (version-agnostic via adapter)
	if name := recordData.GetName(); name != "" {
		annotations[ManifestKeyName] = name
	}

	if version := recordData.GetVersion(); version != "" {
		annotations[ManifestKeyVersion] = version
	}

	// Lifecycle metadata
	if schemaVersion := recordData.GetSchemaVersion(); schemaVersion != "" {
		annotations[ManifestKeySchemaVersion] = schemaVersion
	}

	if createdAt := recordData.GetCreatedAt(); createdAt != "" {
		annotations[ManifestKeyCreatedAt] = createdAt
	}

	// Versioning (v1 specific)
	if previousCid := recordData.GetPreviousRecordCid(); previousCid != "" {
		annotations[ManifestKeyPreviousCid] = previousCid
	}

	// Custom annotations from record data -> manifest custom annotations
	if customAnnotations := recordData.GetAnnotations(); len(customAnnotations) > 0 {
		for key, value := range customAnnotations {
			annotations[ManifestKeyCustomPrefix+key] = value
		}
	}

	return annotations
}

// parseManifestAnnotations extracts structured metadata from manifest annotations.
//
//nolint:cyclop // Function handles multiple metadata extraction paths with justified complexity
func parseManifestAnnotations(annotations map[string]string) *corev1.RecordMeta {
	recordMeta := &corev1.RecordMeta{
		Annotations: make(map[string]string),
	}

	// Set fallback schema version first for error recovery scenarios
	recordMeta.SchemaVersion = FallbackSchemaVersion

	if annotations == nil {
		return recordMeta
	}

	// Extract schema version from stored data (override fallback if present)
	if schemaVersion := annotations[ManifestKeySchemaVersion]; schemaVersion != "" {
		recordMeta.SchemaVersion = schemaVersion
	}

	// Extract created time from stored data (no more empty strings!)
	if createdAt := annotations[ManifestKeyCreatedAt]; createdAt != "" {
		recordMeta.CreatedAt = createdAt
	}

	// Copy structured metadata into annotations for easy access
	// Core identity - these will be easily accessible to consumers
	if name := annotations[ManifestKeyName]; name != "" {
		recordMeta.Annotations[MetadataKeyName] = name
	}

	if version := annotations[ManifestKeyVersion]; version != "" {
		recordMeta.Annotations[MetadataKeyVersion] = version
	}

	if oasfVersion := annotations[ManifestKeyOASFVersion]; oasfVersion != "" {
		recordMeta.Annotations[MetadataKeyOASFVersion] = oasfVersion
	}

	// Versioning information
	if previousCid := annotations[ManifestKeyPreviousCid]; previousCid != "" {
		recordMeta.Annotations[MetadataKeyPreviousCid] = previousCid
	}

	// Custom annotations (those with our custom prefix) - clean namespace
	for key, value := range annotations {
		if after, ok := strings.CutPrefix(key, ManifestKeyCustomPrefix); ok {
			customKey := after
			recordMeta.Annotations[customKey] = value
		}
	}

	return recordMeta
}

// This file defines the complete metadata schema for OCI annotations.
// It serves as the single source of truth for all annotation keys used
// in manifest and descriptor annotations for record storage.

const (
	// Used for dir-specific annotations.
	manifestDirObjectKeyPrefix = "org.agntcy.dir"
	manifestDirObjectTypeKey   = manifestDirObjectKeyPrefix + "/type"

	// THESE ARE THE SOURCE OF TRUTH for field names.

	// Core Identity (simple keys).
	MetadataKeyName        = "name"
	MetadataKeyVersion     = "version"
	MetadataKeyOASFVersion = "oasf-version"
	MetadataKeyCid         = "cid"

	// Lifecycle (simple keys).
	MetadataKeySchemaVersion = "schema-version"
	MetadataKeyCreatedAt     = "created-at"

	// Versioning (simple keys).
	MetadataKeyPreviousCid = "previous-cid"

	// Derived from MetadataKey constants to ensure consistency.

	// Core Identity (derived from MetadataKey constants).
	ManifestKeyName        = manifestDirObjectKeyPrefix + "/" + MetadataKeyName
	ManifestKeyVersion     = manifestDirObjectKeyPrefix + "/" + MetadataKeyVersion
	ManifestKeyOASFVersion = manifestDirObjectKeyPrefix + "/" + MetadataKeyOASFVersion
	ManifestKeyCid         = manifestDirObjectKeyPrefix + "/" + MetadataKeyCid

	// Lifecycle Metadata (mixed: some derived, some standalone).
	ManifestKeySchemaVersion = manifestDirObjectKeyPrefix + "/" + MetadataKeySchemaVersion
	ManifestKeyCreatedAt     = manifestDirObjectKeyPrefix + "/" + MetadataKeyCreatedAt

	// Versioning & Linking (standalone - no simple key equivalents).
	ManifestKeyPreviousCid = manifestDirObjectKeyPrefix + "/" + MetadataKeyPreviousCid

	// Custom annotations prefix.
	ManifestKeyCustomPrefix = manifestDirObjectKeyPrefix + "/custom."

	// Fallback values for error recovery scenarios.
	// Used when parsing corrupted storage, legacy records, or external modifications.
	FallbackSchemaVersion = "0.7.0"
)
