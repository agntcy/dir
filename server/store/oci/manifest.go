package oci

import (
	"fmt"
	"path"

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
	digest, err := corev1.ConvertCIDToDigest(obj.GetCid())
	if err != nil {
		return ocispec.Manifest{}, fmt.Errorf("failed to convert data CID to digest: %w", err)
	}

	// Build link descriptors
	var linkDescriptors []ocispec.Descriptor
	for _, link := range obj.Links {
		digest, err := corev1.ConvertCIDToDigest(link.GetCid())
		if err != nil {
			return ocispec.Manifest{}, fmt.Errorf("failed to build link descriptor: %w", err)
		}

		linkDescriptors = append(linkDescriptors, ocispec.Descriptor{
			Digest:      digest,
			Size:        int64(link.GetSize()),
			Annotations: link.GetAnnotations(),
		})
	}

	return ocispec.Manifest{
		Versioned: imagespecs.Versioned{
			SchemaVersion: int(oras.PackManifestVersion1_1),
		},
		MediaType:    ocispec.MediaTypeImageManifest,
		ArtifactType: manifestArtifactType,
		Config: ocispec.Descriptor{
			Digest:      digest,
			Size:        int64(obj.Size),
			Annotations: obj.Annotations,
		},
		Layers: linkDescriptors,
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

	// Create object
	var object storev1.Object

	// Build object from config descriptor
	dataCID, err := corev1.ConvertDigestToCID(manifest.Config.Digest)
	if err != nil {
		return nil, fmt.Errorf("failed to convert manifest config digest to CID: %w", err)
	}
	object.Cid = dataCID
	object.Size = uint64(manifest.Config.Size)
	object.Annotations = manifest.Config.Annotations

	// Add links from layer descriptors
	for _, layer := range manifest.Layers {
		var linkObj storev1.ObjectLink
		linkCID, err := corev1.ConvertDigestToCID(layer.Digest)
		if err != nil {
			return nil, fmt.Errorf("failed to convert manifest layer digest to CID: %w", err)
		}
		linkObj.Cid = linkCID
		linkObj.Size = uint64(layer.Size)
		linkObj.Annotations = layer.Annotations
		object.Links = append(object.Links, &linkObj)
	}

	return &object, nil
}
