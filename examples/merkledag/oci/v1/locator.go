package v1

import (
	"encoding/json"
	"fmt"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// LocatorHandler handles locator entities
type LocatorHandler struct{}

func (h *LocatorHandler) MediaType() string {
	return "application/vnd.agntcy.record.locator.v1+json"
}

func (h *LocatorHandler) GetEntities(record *oasfv1.Record) []interface{} {
	entities := make([]interface{}, len(record.GetLocators()))
	for i, locator := range record.GetLocators() {
		entities[i] = locator
	}
	return entities
}

func (h *LocatorHandler) ToLayer(entity interface{}) (ocispec.Descriptor, error) {
	locator, ok := entity.(*oasfv1.Locator)
	if !ok {
		return ocispec.Descriptor{}, fmt.Errorf("entity is not a locator")
	}

	entityBytes, err := json.Marshal(locator)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal locator: %w", err)
	}

	desc := ocispec.Descriptor{
		MediaType: h.MediaType(),
		Digest:    digest.FromBytes(entityBytes),
		Size:      int64(len(entityBytes)),
		Data:      entityBytes,
	}

	return desc, nil
}

func (h *LocatorHandler) FromLayer(descriptor ocispec.Descriptor) (interface{}, error) {
	if len(descriptor.Data) == 0 {
		return nil, fmt.Errorf("descriptor data is empty")
	}

	var locator oasfv1.Locator
	if err := json.Unmarshal(descriptor.Data, &locator); err != nil {
		return nil, fmt.Errorf("failed to unmarshal locator: %w", err)
	}

	return &locator, nil
}

func (h *LocatorHandler) AppendToRecord(record *oasfv1.Record, entity interface{}) {
	if locator, ok := entity.(*oasfv1.Locator); ok {
		record.Locators = append(record.Locators, locator)
	}
}
