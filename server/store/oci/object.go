package oci

import (
	"context"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	annotationCreatedAt = "org.agntcy.dir/object.created_at"
	annotationType      = "org.agntcy.dir/object.type"
)

// packageObject converts an Object and data descriptor into an OCI manifest.
func (s *store) packageObject(ctx context.Context, object *storev1.Object, dataDesc ocispec.Descriptor) (*ocispec.Manifest, error) {
	if object == nil {
		return nil, fmt.Errorf("object metadata cannot be nil")
	}

	// Create OCI manifest
	manifest := &ocispec.Manifest{
		Versioned: imagespec.Versioned{
			SchemaVersion: 2,
		},
		MediaType: ocispec.MediaTypeImageManifest,
		// Set data information in config descriptor
		Config: ocispec.Descriptor{
			// Set data descriptor fields
			MediaType: dataDesc.MediaType,
			Size:      dataDesc.Size,
			Digest:    dataDesc.Digest,
			// Set object annotations as part of data descriptor
			Annotations: object.Annotations,
		},
		Layers:      []ocispec.Descriptor{},
		Annotations: map[string]string{},
	}

	// Store Object fields in annotations
	if object.CreatedAt != nil {
		manifest.Annotations[annotationCreatedAt] = *object.CreatedAt
	}
	if object.Type != nil {
		manifest.Annotations[annotationType] = *object.Type
	}

	// Store links as layers
	if len(object.Links) > 0 {
		// Fetch link descriptors
		linkDescs, err := s.descLookupMany(ctx, object.Links)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch link descriptors: %w", err)
		}

		// Append link descriptors as layers, preserve order
		for _, ref := range object.Links {
			desc, ok := linkDescs[ref.Cid]
			if !ok {
				return nil, fmt.Errorf("link descriptor not found for CID: %s", ref.Cid)
			}
			manifest.Layers = append(manifest.Layers, desc)
		}
	} else {
		// No links, set empty layers
		manifest.Layers = append(manifest.Layers, ocispec.DescriptorEmptyJSON)
	}

	// Store parent as subject
	if object.Parent != nil {
		// Fetch parent descriptor
		linkDescs, err := s.descLookupMany(ctx, []*storev1.ObjectRef{object.Parent})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch link descriptors: %w", err)
		}

		// Set subject if found
		if desc, ok := linkDescs[object.Parent.Cid]; ok {
			manifest.Subject = &desc
		} else {
			return nil, fmt.Errorf("parent descriptor not found for CID: %s", object.Parent.Cid)
		}
	}

	return manifest, nil
}

// unpackageObject extracts Object and data descriptor from an OCI manifest.
// Returns the reconstructed Object and the data layer descriptor.
func (s *store) unpackageObject(ctx context.Context, manifest *ocispec.Manifest) (*storev1.Object, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest cannot be nil")
	}

	// Reconstruct Object from annotations
	object := &storev1.Object{
		Annotations: make(map[string]string),
	}

	// Parse created_at
	if val, ok := manifest.Annotations[annotationCreatedAt]; ok {
		object.CreatedAt = &val
	}

	// Parse type
	if val, ok := manifest.Annotations[annotationType]; ok {
		object.Type = &val
	}

	// Parse data size (from config descriptor)
	object.Size = uint64(manifest.Config.Size)
	object.Annotations = manifest.Config.Annotations

	// Reconstruct links from layers
	if len(manifest.Layers) > 0 {
		object.Links = make([]*storev1.ObjectRef, 0, len(manifest.Layers))
		for _, layer := range manifest.Layers {
			// Skip empty layer
			if layer.Digest == ocispec.DescriptorEmptyJSON.Digest {
				continue
			}

			// Convert OCI digest to CID
			cid, err := corev1.ConvertDigestToCID(layer.Digest)
			if err != nil {
				return nil, fmt.Errorf("failed to convert layer digest to CID: %w", err)
			}
			object.Links = append(object.Links, &storev1.ObjectRef{Cid: cid})
		}
	}

	// Reconstruct parent from subject
	if manifest.Subject != nil {
		cid, err := corev1.ConvertDigestToCID(manifest.Subject.Digest)
		if err != nil {
			return nil, fmt.Errorf("failed to convert subject digest to CID: %w", err)
		}
		object.Parent = &storev1.ObjectRef{Cid: cid}
	}

	return object, nil
}
