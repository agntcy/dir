package v1

import (
	"encoding/json"
	"fmt"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// DomainHandler handles domain entities
type DomainHandler struct{}

func (h *DomainHandler) MediaType() string {
	return "application/vnd.agntcy.record.domain.v1+json"
}

func (h *DomainHandler) GetEntities(record *oasfv1.Record) []interface{} {
	entities := make([]interface{}, len(record.GetDomains()))
	for i, domain := range record.GetDomains() {
		entities[i] = domain
	}
	return entities
}

func (h *DomainHandler) ToLayer(entity interface{}) (ocispec.Descriptor, error) {
	domain, ok := entity.(*oasfv1.Domain)
	if !ok {
		return ocispec.Descriptor{}, fmt.Errorf("entity is not a domain")
	}

	entityBytes, err := json.Marshal(domain)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal domain: %w", err)
	}

	desc := ocispec.Descriptor{
		MediaType: h.MediaType(),
		Digest:    digest.FromBytes(entityBytes),
		Size:      int64(len(entityBytes)),
		Data:      entityBytes,
	}

	if desc.Annotations == nil {
		desc.Annotations = make(map[string]string)
	}
	if name := domain.GetName(); name != "" {
		desc.Annotations["org.agntcy.domain.name"] = name
	}

	return desc, nil
}

func (h *DomainHandler) FromLayer(descriptor ocispec.Descriptor) (interface{}, error) {
	if len(descriptor.Data) == 0 {
		return nil, fmt.Errorf("descriptor data is empty")
	}

	var domain oasfv1.Domain
	if err := json.Unmarshal(descriptor.Data, &domain); err != nil {
		return nil, fmt.Errorf("failed to unmarshal domain: %w", err)
	}

	return &domain, nil
}

func (h *DomainHandler) AppendToRecord(record *oasfv1.Record, entity interface{}) {
	if domain, ok := entity.(*oasfv1.Domain); ok {
		record.Domains = append(record.Domains, domain)
	}
}
