package v1

import (
	"encoding/json"
	"fmt"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ModuleHandler handles module entities
type ModuleHandler struct{}

func (h *ModuleHandler) MediaType() string {
	return "application/vnd.agntcy.record.module.v1+json"
}

func (h *ModuleHandler) GetEntities(record *oasfv1.Record) []interface{} {
	entities := make([]interface{}, len(record.GetModules()))
	for i, module := range record.GetModules() {
		entities[i] = module
	}
	return entities
}

func (h *ModuleHandler) ToLayer(entity interface{}) (ocispec.Descriptor, error) {
	module, ok := entity.(*oasfv1.Module)
	if !ok {
		return ocispec.Descriptor{}, fmt.Errorf("entity is not a module")
	}

	entityBytes, err := json.Marshal(module)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal module: %w", err)
	}

	desc := ocispec.Descriptor{
		MediaType: h.MediaType(),
		Digest:    digest.FromBytes(entityBytes),
		Size:      int64(len(entityBytes)),
		Data:      entityBytes,
	}

	return desc, nil
}

func (h *ModuleHandler) FromLayer(descriptor ocispec.Descriptor) (interface{}, error) {
	if len(descriptor.Data) == 0 {
		return nil, fmt.Errorf("descriptor data is empty")
	}

	var module oasfv1.Module
	if err := json.Unmarshal(descriptor.Data, &module); err != nil {
		return nil, fmt.Errorf("failed to unmarshal module: %w", err)
	}

	return &module, nil
}

func (h *ModuleHandler) AppendToRecord(record *oasfv1.Record, entity interface{}) {
	if module, ok := entity.(*oasfv1.Module); ok {
		record.Modules = append(record.Modules, module)
	}
}
