package v1

import (
	"encoding/json"
	"fmt"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// MetadataHandler handles record metadata (config layer)
type MetadataHandler struct{}

func (h *MetadataHandler) MediaType() string {
	return "application/vnd.agntcy.record.config.v1+json"
}

func (h *MetadataHandler) GetEntities(record *oasfv1.Record) []interface{} {
	return []interface{}{record}
}

func (h *MetadataHandler) ToLayer(entity interface{}) (ocispec.Descriptor, error) {
	record, ok := entity.(*oasfv1.Record)
	if !ok {
		return ocispec.Descriptor{}, fmt.Errorf("entity is not a record")
	}

	metadata := &oasfv1.Record{
		Name:          record.GetName(),
		Version:       record.GetVersion(),
		SchemaVersion: record.GetSchemaVersion(),
		Authors:       record.GetAuthors(),
		Description:   record.GetDescription(),
	}

	entityBytes, err := json.Marshal(metadata)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return ocispec.Descriptor{
		MediaType:   h.MediaType(),
		Digest:      digest.FromBytes(entityBytes),
		Size:        int64(len(entityBytes)),
		Data:        entityBytes,
		Annotations: record.GetAnnotations(),
	}, nil
}

func (h *MetadataHandler) FromLayer(descriptor ocispec.Descriptor) (interface{}, error) {
	if len(descriptor.Data) == 0 {
		return nil, fmt.Errorf("descriptor data is empty")
	}

	metadata := &oasfv1.Record{}
	if err := json.Unmarshal(descriptor.Data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &oasfv1.Record{
		Name:          metadata.Name,
		Version:       metadata.Version,
		SchemaVersion: metadata.SchemaVersion,
		Authors:       metadata.Authors,
		Description:   metadata.Description,
		Annotations:   descriptor.Annotations,
	}, nil
}

func (h *MetadataHandler) AppendToRecord(record *oasfv1.Record, entity interface{}) {
	if metadata, ok := entity.(*oasfv1.Record); ok {
		record.Name = metadata.Name
		record.Version = metadata.Version
		record.SchemaVersion = metadata.SchemaVersion
		record.Authors = metadata.Authors
		record.Description = metadata.Description
		record.Annotations = metadata.Annotations
	}
}
