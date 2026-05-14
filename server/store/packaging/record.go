// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// This file defines the complete schema for Record <-> Manifest packaging.
// It serves as the single source of truth for construction, validation,
// and parsing of Record objects.

package packaging

import (
	"context"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev2 "github.com/agntcy/dir/api/store/v2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/server/types/adapters"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

const (
	// RecordMediaType is the media type used for records.
	// This is the media type that clients should use when
	// dealing with Record objects.
	RecordMediaType = "application/vnd.agntcy.dir.v1+json"

	// Annotations for record manifests.
	recordAnnotationPrefix        = "org.agntcy.dir.record"
	recordNameAnnotation          = recordAnnotationPrefix + "/name"
	recordVersionAnnotation       = recordAnnotationPrefix + "/version"
	recordSchemaVersionAnnotation = recordAnnotationPrefix + "/schema-version"
	recordCreatedAtAnnotation     = recordAnnotationPrefix + "/created-at"
	recordPreviousCidAnnotation   = recordAnnotationPrefix + "/previous-cid"
	recordCustomAnnotationPrefix  = recordAnnotationPrefix + "/annotation."
)

type record struct{}

func init() {
	RegisterPacker(RecordMediaType, &record{})
}

func (r *record) Pack(ctx context.Context, repo *remote.Repository, obj *storev2.Object) (*ocispec.Manifest, error) {
	// Convert object to a record
	record, err := corev1.UnmarshalRecord(obj.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal object data into record: %w", err)
	}

	// Get record data via adapter
	recordData, err := adapters.NewRecordAdapter(record).GetRecordData()
	if err != nil {
		return nil, fmt.Errorf("failed to read record data: %w", err)
	}

	// If there is previous record CID, fetch its descriptor to add as subject
	// The subject can only be a manifest, faily otherwise
	var subject *ocispec.Descriptor

	if prevCID := recordData.GetPreviousRecordCid(); prevCID != "" {
		prevDesc, err := repo.Manifests().Resolve(ctx, prevCID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve previous record CID: %w", err)
		}

		// Set subject
		subject = &prevDesc
	}

	// Marshal the record using canonical JSON marshaling first
	// This ensures consistent bytes for both CID calculation and storage
	recordBytes, err := record.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record: %w", err)
	}

	// Push the record bytes as a blob
	recordDesc, err := oras.PushBytes(ctx, repo, "application/json", recordBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to push record bytes: %w", err)
	}

	return &ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: int(oras.PackManifestVersion1_1),
		},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: RecordMediaType,
		Config:       recordDesc,
		Layers:       []ocispec.Descriptor{ocispec.DescriptorEmptyJSON},
		Subject:      subject,
		Annotations:  r.extractManifestAnnotations(recordData),
	}, nil
}

func (r *record) Unpack(ctx context.Context, repo *remote.Repository, manifest *ocispec.Manifest) (*storev2.Object, error) {
	// Pull the config blob which contains the record bytes
	reader, err := repo.Fetch(ctx, manifest.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch config content: %w", err)
	}
	defer reader.Close()

	recordBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read config content: %w", err)
	}

	// Validate record
	_, err = corev1.UnmarshalRecord(recordBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config bytes into record: %w", err)
	}

	return &storev2.Object{
		MediaType: RecordMediaType,
		Size:      uint64(len(recordBytes)),
		Data:      recordBytes,
	}, nil
}

// extractManifestAnnotations extracts manifest annotations from record using adapter pattern.
//
//nolint:cyclop // Function handles multiple annotation types with justified complexity
func (r *record) extractManifestAnnotations(recordData types.RecordData) map[string]string {
	annotations := make(map[string]string)

	// Add annotations for manifest
	if schemaVersion := recordData.GetSchemaVersion(); schemaVersion != "" {
		annotations[recordSchemaVersionAnnotation] = schemaVersion
	}

	if name := recordData.GetName(); name != "" {
		annotations[recordNameAnnotation] = name
	}

	if version := recordData.GetVersion(); version != "" {
		annotations[recordVersionAnnotation] = version
	}

	if schemaVersion := recordData.GetSchemaVersion(); schemaVersion != "" {
		annotations[recordSchemaVersionAnnotation] = schemaVersion
	}

	if createdAt := recordData.GetCreatedAt(); createdAt != "" {
		annotations[recordCreatedAtAnnotation] = createdAt
	}

	if previousCid := recordData.GetPreviousRecordCid(); previousCid != "" {
		annotations[recordPreviousCidAnnotation] = previousCid
	}

	// Custom annotations from record data -> manifest custom annotations
	if customAnnotations := recordData.GetAnnotations(); len(customAnnotations) > 0 {
		for key, value := range customAnnotations {
			annotations[recordCustomAnnotationPrefix+key] = value
		}
	}

	return annotations
}
