// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
)

// aiCatalogModuleName is the OASF module whose document lives on the artifact.
// It matches catalog/v1.AICatalogModuleName. This package cannot import catalog
// (that import cycles through core/v1).
const aiCatalogModuleName = "integration/ai_catalog"

type v1Adapter struct {
	cid    string
	record *typesv1.Record
}

func newV1Adapter(cid string, record *typesv1.Record) coretypes.Record {
	return &v1Adapter{
		cid:    cid,
		record: record,
	}
}

func (v *v1Adapter) GetCid() string {
	return v.cid
}

func (v *v1Adapter) GetAnnotations() map[string]string {
	return v.record.GetAnnotations()
}

func (v *v1Adapter) GetAuthors() []string {
	return v.record.GetAuthors()
}

func (v *v1Adapter) GetCreatedAt() string {
	return v.record.GetCreatedAt()
}

func (v *v1Adapter) GetDescription() string {
	return v.record.GetDescription()
}

func (v *v1Adapter) GetDomains() []coretypes.Domain {
	domains := make([]coretypes.Domain, 0, len(v.record.GetDomains()))
	for _, d := range v.record.GetDomains() {
		domains = append(domains, &domain{
			Annotations: d.GetAnnotations(),
			Name:        d.GetName(),
			ID:          uint64(d.GetId()),
		})
	}

	return domains
}

func (v *v1Adapter) GetLocators() []coretypes.Locator {
	locators := make([]coretypes.Locator, 0, len(v.record.GetLocators()))
	for _, l := range v.record.GetLocators() {
		var url string
		if urls := l.GetUrls(); len(urls) > 0 {
			url = urls[0] // Assuming we take the first URL if multiple are present
		}

		locators = append(locators, &locator{
			Annotations: l.GetAnnotations(),
			Type:        l.GetType(),
			URL:         url,
		})
	}

	return locators
}

func (v *v1Adapter) GetModules() []coretypes.Module {
	modules := make([]coretypes.Module, 0, len(v.record.GetModules()))
	for _, m := range v.record.GetModules() {
		artifactMediaType := ""

		var artifact *coretypes.ModuleArtifact

		if descriptor := m.GetArtifact(); descriptor != nil {
			artifactMediaType = descriptor.GetMediaType()
			// Only an AI Catalog module carries its document on the artifact.
			// Other modules keep the media type; their payload stays in data.
			if m.GetName() == aiCatalogModuleName {
				artifact = moduleArtifact(descriptor)
			}
		}

		modules = append(modules, &module{
			Annotations:       m.GetAnnotations(),
			Name:              m.GetName(),
			ID:                uint64(m.GetId()),
			Data:              m.GetData().AsMap(),
			ArtifactMediaType: artifactMediaType,
			Artifact:          artifact,
		})
	}

	return modules
}

// moduleArtifact copies the descriptor payload. An empty descriptor is absent.
func moduleArtifact(descriptor *typesv1.Descriptor) *coretypes.ModuleArtifact {
	if descriptor == nil {
		return nil
	}

	artifact := &coretypes.ModuleArtifact{
		URLs: append([]string(nil), descriptor.GetUrls()...),
		Data: append([]byte(nil), descriptor.GetData()...),
	}
	if payload := descriptor.GetJson(); payload != nil {
		artifact.JSON = payload.AsMap()
	}

	if len(artifact.URLs) == 0 && len(artifact.Data) == 0 && artifact.JSON == nil {
		return nil
	}

	return artifact
}

func (v *v1Adapter) GetName() string {
	return v.record.GetName()
}

func (v *v1Adapter) GetPreviousRecordCid() string {
	return ""
}

func (v *v1Adapter) GetSchemaVersion() string {
	return v.record.GetSchemaVersion()
}

func (v *v1Adapter) GetSkills() []coretypes.Skill {
	skills := make([]coretypes.Skill, 0, len(v.record.GetSkills()))
	for _, s := range v.record.GetSkills() {
		skills = append(skills, &skill{
			Annotations: s.GetAnnotations(),
			Name:        s.GetName(),
			ID:          uint64(s.GetId()),
		})
	}

	return skills
}

func (v *v1Adapter) GetVersion() string {
	return v.record.GetVersion()
}
