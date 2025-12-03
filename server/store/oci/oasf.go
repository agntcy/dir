package oci

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	// Annotation keys for storing RecordMeta in OCI manifest
	annotationCreatedAt = "agntcy.dir.record.created_at"
	annotationType      = "agntcy.dir.record.type"
)

// packageRecord converts a RecordMeta and data descriptor into an OCI manifest.
// The manifest uses layers[0] for the data blob and annotations for metadata.
func (s *store) packageRecord(ctx context.Context, record *corev1.RecordMeta, dataDesc ocispec.Descriptor) (*ocispec.Manifest, error) {
	if record == nil {
		return nil, fmt.Errorf("record metadata cannot be nil")
	}

	// Create OCI manifest
	manifest := &ocispec.Manifest{
		MediaType:   ocispec.MediaTypeImageManifest,
		Config:      dataDesc,
		Layers:      []ocispec.Descriptor{},
		Annotations: map[string]string{},
	}

	// Store RecordMeta fields in annotations
	// manifest.Annotations[annotationCID] = record.Cid
	manifest.Annotations[annotationCreatedAt] = record.CreatedAt
	manifest.Annotations[annotationType] = record.Type

	// Copy user annotations
	for key, value := range record.Annotations {
		if manifest.Config.Annotations == nil {
			manifest.Config.Annotations = make(map[string]string)
		}
		manifest.Config.Annotations[key] = value
	}

	// Write links as layers
	if len(record.Links) > 0 {
		// Fetch link descriptors
		linkDescs, err := s.descLookupMany(ctx, record.Links)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch link descriptors: %w", err)
		}

		for _, linkDesc := range linkDescs {
			manifest.Layers = append(manifest.Layers, linkDesc)
		}
	}

	// Write parent if present
	if record.Parent != nil {
		// Fetch link descriptor
		linkDescs, err := s.descLookupMany(ctx, []*corev1.RecordRef{record.Parent})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch link descriptors: %w", err)
		}

		if desc, ok := linkDescs[record.Parent.Cid]; ok {
			manifest.Subject = &desc
		} else {
			return nil, fmt.Errorf("parent descriptor not found for CID: %s", record.Parent.Cid)
		}
	}

	return manifest, nil
}

// unpackageRecord extracts RecordMeta and data descriptor from an OCI manifest.
// Returns the reconstructed RecordMeta and the data layer descriptor.
func (s *store) unpackageRecord(ctx context.Context, manifest *ocispec.Manifest) (*corev1.RecordMeta, ocispec.Descriptor, error) {
	if manifest == nil {
		return nil, ocispec.Descriptor{}, fmt.Errorf("manifest cannot be nil")
	}

	if manifest.Annotations == nil {
		return nil, ocispec.Descriptor{}, fmt.Errorf("manifest annotations are required")
	}

	// Reconstruct RecordMeta from annotations
	record := &corev1.RecordMeta{
		Annotations: make(map[string]string),
	}

	record.CreatedAt = manifest.Annotations[annotationCreatedAt]
	record.Type = manifest.Annotations[annotationType]

	// Parse size
	record.Size = uint64(manifest.Config.Size)

	// Extract user annotations from config descriptor
	if manifest.Config.Annotations != nil {
		for key, value := range manifest.Config.Annotations {
			record.Annotations[key] = value
		}
	}

	// Reconstruct links from layers
	if len(manifest.Layers) > 0 {
		record.Links = make([]*corev1.RecordRef, 0, len(manifest.Layers))
		for _, layer := range manifest.Layers {
			// Convert OCI digest to CID
			cid, err := corev1.ConvertDigestToCID(layer.Digest)
			if err != nil {
				return nil, ocispec.Descriptor{}, fmt.Errorf("failed to convert layer digest to CID: %w", err)
			}
			record.Links = append(record.Links, &corev1.RecordRef{
				Cid: cid,
			})
		}
	}

	// Reconstruct parent from subject
	if manifest.Subject != nil {
		cid, err := corev1.ConvertDigestToCID(manifest.Subject.Digest)
		if err != nil {
			return nil, ocispec.Descriptor{}, fmt.Errorf("failed to convert subject digest to CID: %w", err)
		}
		record.Parent = &corev1.RecordRef{
			Cid: cid,
		}
	}

	// Return the config descriptor as data descriptor
	dataDesc := manifest.Config

	return record, dataDesc, nil
}
