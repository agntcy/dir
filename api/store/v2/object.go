package v2

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

func NewDescriptor(desc ocispec.Descriptor) *ObjectDescriptor {
	return &ObjectDescriptor{
		Digest:       desc.Digest.String(),
		MediaType:    desc.MediaType,
		ArtifactType: desc.ArtifactType,
		Size:         uint64(desc.Size),
		Urls:         desc.URLs,
		Annotations:  desc.Annotations,
	}
}
