package v1

import (
	"encoding/json"
	"fmt"

	oasfv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// SkillHandler handles skill entities
type SkillHandler struct{}

func (h *SkillHandler) MediaType() string {
	return "application/vnd.agntcy.record.skill.v1+json"
}

func (h *SkillHandler) GetEntities(record *oasfv1.Record) []interface{} {
	entities := make([]interface{}, len(record.GetSkills()))
	for i, skill := range record.GetSkills() {
		entities[i] = skill
	}
	return entities
}

func (h *SkillHandler) ToLayer(entity interface{}) (ocispec.Descriptor, error) {
	skill, ok := entity.(*oasfv1.Skill)
	if !ok {
		return ocispec.Descriptor{}, fmt.Errorf("entity is not a skill")
	}

	entityBytes, err := json.Marshal(skill)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal skill: %w", err)
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
	if name := skill.GetName(); name != "" {
		desc.Annotations["org.agntcy.skill.name"] = name
	}

	return desc, nil
}

func (h *SkillHandler) FromLayer(descriptor ocispec.Descriptor) (interface{}, error) {
	if len(descriptor.Data) == 0 {
		return nil, fmt.Errorf("descriptor data is empty")
	}

	var skill oasfv1.Skill
	if err := json.Unmarshal(descriptor.Data, &skill); err != nil {
		return nil, fmt.Errorf("failed to unmarshal skill: %w", err)
	}

	return &skill, nil
}

func (h *SkillHandler) AppendToRecord(record *oasfv1.Record, entity interface{}) {
	if skill, ok := entity.(*oasfv1.Skill); ok {
		record.Skills = append(record.Skills, skill)
	}
}
