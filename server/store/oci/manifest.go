package oci

import (
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const (
	annotationPrefix     = "org.agntcy.oasf."
	annotationSchemaType = annotationPrefix + "schema.type"
	annotationSchemaVer  = annotationPrefix + "schema.version"
	annotationCreatedAt  = annotationPrefix + "schema.created_at"
	annotationSchemaFmt  = annotationPrefix + "schema.format"
)

// ObjectToManifest converts an OASF Object to an OCI Manifest
func ObjectToManifest(obj *corev1.Object) (ocispec.Manifest, error) {
	if obj == nil {
		return ocispec.Manifest{}, fmt.Errorf("object cannot be nil")
	}

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
	}

	// Build config descriptor from object data and schema
	if obj.Data != nil {
		if obj.Data.Cid == "" {
			return ocispec.Manifest{}, fmt.Errorf("object data CID cannot be empty")
		}
		configAnnotations := buildAnnotations(obj)
		// Convert CID to OCI digest
		digest, err := corev1.ConvertCIDToDigest(obj.Data.Cid)
		if err != nil {
			return ocispec.Manifest{}, fmt.Errorf("failed to convert data CID to digest: %w", err)
		}
		manifest.Config = ocispec.Descriptor{
			MediaType:   "application/octet-stream",
			Digest:      digest,
			Size:        int64(obj.Data.Size),
			Annotations: configAnnotations,
		}
	}

	// Build layers from links
	if len(obj.Links) > 0 {
		manifest.Layers = make([]ocispec.Descriptor, 0, len(obj.Links))
		for i, link := range obj.Links {
			if link.Data != nil {
				if link.Data.Cid == "" {
					return ocispec.Manifest{}, fmt.Errorf("link[%d] data CID cannot be empty", i)
				}
				layerAnnotations := buildAnnotations(link)
				// Convert CID to OCI digest
				digest, err := corev1.ConvertCIDToDigest(link.Data.Cid)
				if err != nil {
					return ocispec.Manifest{}, fmt.Errorf("failed to convert link[%d] CID to digest: %w", i, err)
				}
				layer := ocispec.Descriptor{
					MediaType:   "application/octet-stream",
					Digest:      digest,
					Size:        int64(link.Data.Size),
					Annotations: layerAnnotations,
				}
				manifest.Layers = append(manifest.Layers, layer)
			}
		}
	}

	return manifest, nil
}

// ManifestToObject converts an OCI Manifest to an OASF Object
func ManifestToObject(manifest *ocispec.Manifest) (*corev1.Object, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest cannot be nil")
	}

	obj := &corev1.Object{}

	// Extract object data and schema from config descriptor
	if manifest.Config.Digest != "" {
		// Convert OCI digest to CID
		cid, err := corev1.ConvertDigestToCID(manifest.Config.Digest)
		if err != nil {
			return nil, fmt.Errorf("failed to convert config digest to CID: %w", err)
		}
		obj.Data = &corev1.Object{
			Cid:  cid,
			Size: uint64(manifest.Config.Size),
		}

		// Extract schema and annotations from config
		obj.Schema = extractSchema(manifest.Config.Annotations)
		obj.Annotations = extractUserAnnotations(manifest.Config.Annotations)
		obj.CreatedAt = manifest.Config.Annotations[annotationCreatedAt]
	}

	// Extract links from layers
	if len(manifest.Layers) > 0 {
		obj.Links = make([]*corev1.Object, 0, len(manifest.Layers))
		for i, layer := range manifest.Layers {
			// Convert OCI digest to CID
			cid, err := corev1.ConvertDigestToCID(layer.Digest)
			if err != nil {
				return nil, fmt.Errorf("failed to convert layer[%d] digest to CID: %w", i, err)
			}
			link := &corev1.Object{
				Data: &corev1.Object{
					Cid:  cid,
					Size: uint64(layer.Size),
				},
				Schema:      extractSchema(layer.Annotations),
				Annotations: extractUserAnnotations(layer.Annotations),
				CreatedAt:   layer.Annotations[annotationCreatedAt],
			}
			obj.Links = append(obj.Links, link)
		}
	}

	return obj, nil
}

// buildAnnotations builds OCI annotations from an OASF Object
func buildAnnotations(obj *corev1.Object) map[string]string {
	annotations := make(map[string]string)

	// Add schema annotations
	if obj.Schema != nil {
		if obj.Schema.Type != "" {
			annotations[annotationSchemaType] = obj.Schema.Type
		}
		if obj.Schema.Version != "" {
			annotations[annotationSchemaVer] = obj.Schema.Version
		}
		if obj.Schema.Format != "" {
			annotations[annotationSchemaFmt] = obj.Schema.Format
		}
	}

	// Add created_at annotation
	if obj.CreatedAt != "" {
		annotations[annotationCreatedAt] = obj.CreatedAt
	}

	// Add user annotations
	for key, value := range obj.Annotations {
		annotations[key] = value
	}

	return annotations
}

// extractSchema extracts ObjectSchema from OCI annotations
func extractSchema(annotations map[string]string) *corev1.ObjectSchema {
	if annotations == nil {
		return nil
	}

	schema := &corev1.ObjectSchema{}
	hasSchema := false

	if schemaType, ok := annotations[annotationSchemaType]; ok {
		schema.Type = schemaType
		hasSchema = true
	}
	if version, ok := annotations[annotationSchemaVer]; ok {
		schema.Version = version
		hasSchema = true
	}
	if format, ok := annotations[annotationSchemaFmt]; ok {
		schema.Format = format
		hasSchema = true
	}

	if !hasSchema {
		return nil
	}

	return schema
}

// extractUserAnnotations extracts user annotations (non-OASF-prefixed) from OCI annotations
func extractUserAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return nil
	}

	userAnnotations := make(map[string]string)
	for key, value := range annotations {
		// Skip OASF system annotations
		if key != annotationSchemaType &&
			key != annotationSchemaVer &&
			key != annotationCreatedAt &&
			key != annotationSchemaFmt {
			userAnnotations[key] = value
		}
	}

	if len(userAnnotations) == 0 {
		return nil
	}

	return userAnnotations
}
