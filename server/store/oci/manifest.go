package oci

import (
	"fmt"
	"path"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	storev1 "github.com/agntcy/dir/api/store/v1"
	imagespecs "github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
)

var (
	manifestArtifactType = "application/vnd.org.agntcy.dir.Object"
	annotationPrefix     = "org.agntcy.dir"
	annotationCreatedAt  = path.Join(annotationPrefix, "created_at")
	annotationSchemaType = path.Join(annotationPrefix, "schema.type")
	annotationSchemaVer  = path.Join(annotationPrefix, "schema.version")
	annotationSchemaFmt  = path.Join(annotationPrefix, "schema.format")
)

// ObjectToManifest converts DIR Object to an OCI Manifest
func ObjectToManifest(obj *storev1.Object) (ocispec.Manifest, error) {
	// Validate input
	if obj == nil {
		return ocispec.Manifest{}, fmt.Errorf("object cannot be nil")
	}

	// Build data descriptor
	dataDescriptor, err := dataToDescriptor(obj)
	if err != nil {
		return ocispec.Manifest{}, fmt.Errorf("failed to build data config descriptor: %w", err)
	}

	// Build link descriptors
	var linkDescriptors []ocispec.Descriptor
	for _, link := range obj.Links {
		linkDesc, err := dataToDescriptor(link)
		if err != nil {
			return ocispec.Manifest{}, fmt.Errorf("failed to build link descriptor: %w", err)
		}

		linkDescriptors = append(linkDescriptors, linkDesc)
	}

	return ocispec.Manifest{
		Versioned: imagespecs.Versioned{
			SchemaVersion: int(oras.PackManifestVersion1_1),
		},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: manifestArtifactType,
		Config:       dataDescriptor,
		Layers:       linkDescriptors,
	}, nil
}

// ManifestToObject converts an OCI Manifest to DIR Object
func ManifestToObject(manifest *ocispec.Manifest) (*storev1.Object, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest cannot be nil")
	}
	if manifest.ArtifactType != manifestArtifactType {
		return nil, fmt.Errorf("unsupported manifest artifact type: %s", manifest.ArtifactType)
	}

	// Build object from config descriptor
	obj, err := descriptorToData(&manifest.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to build object from manifest config: %w", err)
	}

	// Add links from layer descriptors
	for _, layer := range manifest.Layers {
		linkObj, err := descriptorToData(&layer)
		if err != nil {
			return nil, fmt.Errorf("failed to build link object from manifest layer: %w", err)
		}
		obj.Links = append(obj.Links, linkObj)
	}

	return obj, nil
}

// dataToDescriptor builds OCI descriptor from DIR data object
func dataToDescriptor(obj *storev1.Object) (ocispec.Descriptor, error) {
	// Get data digest
	if obj.Data == nil || obj.Data.Cid == "" {
		return ocispec.Descriptor{}, fmt.Errorf("object data CID cannot be empty")
	}

	digest, err := corev1.ConvertCIDToDigest(obj.Data.Cid)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to convert data CID to digest: %w", err)
	}

	// Build annotations
	annotations := make(map[string]string)

	// Add object annotations
	for key, value := range obj.Annotations {
		if !strings.HasPrefix(key, annotationPrefix) {
			annotations[key] = value
		}
	}

	// Add schema
	if obj.Schema != nil {
		if schemaType := obj.Schema.Type; schemaType != "" {
			annotations[annotationSchemaType] = schemaType
		}
		if version := obj.Schema.Version; version != "" {
			annotations[annotationSchemaVer] = version
		}
		if format := obj.Schema.Format; format != "" {
			annotations[annotationSchemaFmt] = format
		}
	}

	// Add created at
	if createdAt := obj.CreatedAt; createdAt != "" {
		annotations[annotationCreatedAt] = createdAt
	}

	return ocispec.Descriptor{
		Digest:      digest,
		Size:        int64(obj.Size),
		Annotations: annotations,
	}, nil
}

// descriptorToData builds DIR data object from OCI descriptor
func descriptorToData(desc *ocispec.Descriptor) (*storev1.Object, error) {
	// Get CID from digest
	cid, err := corev1.ConvertDigestToCID(desc.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to convert descriptor digest to CID: %w", err)
	}

	// Extract data from descriptor annotations
	annotations := make(map[string]string)
	schema := &storev1.ObjectSchema{}
	createdAt := ""

	// Extract created at
	if desc.Annotations != nil {
		// Extract created at
		if ca, ok := desc.Annotations[annotationCreatedAt]; ok {
			createdAt = ca
		}

		// Extract schema
		if t, ok := desc.Annotations[annotationSchemaType]; ok {
			schema.Type = t
		}
		if v, ok := desc.Annotations[annotationSchemaVer]; ok {
			schema.Version = v
		}
		if f, ok := desc.Annotations[annotationSchemaFmt]; ok {
			schema.Format = f
		}

		// Extract user annotations
		for key, value := range desc.Annotations {
			if !strings.HasPrefix(key, annotationPrefix) {
				annotations[key] = value
			}
		}
	}

	return &storev1.Object{
		Data:        &storev1.ObjectRef{Cid: cid},
		Size:        uint64(desc.Size),
		Annotations: annotations,
		Schema:      schema,
		CreatedAt:   createdAt,
	}, nil
}
