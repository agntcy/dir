// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1alpha2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	coretypes "github.com/agntcy/dir/api/core/types"
)

type v1Alpha2Adapter struct {
	cid    string
	record *typesv1alpha2.Record
}

func newV1Alpha2Adapter(cid string, record *typesv1alpha2.Record) coretypes.Record {
	return &v1Alpha2Adapter{
		cid:    cid,
		record: record,
	}
}

func (v *v1Alpha2Adapter) GetCid() string {
	return v.cid
}

func (v *v1Alpha2Adapter) GetModules() []coretypes.Module {
	modules := make([]coretypes.Module, 0, len(v.record.GetModules()))
	for _, m := range v.record.GetModules() {
		modules = append(modules, &module{
			Annotations: m.GetAnnotations(),
			Name:        m.GetName(),
			ID:          uint64(m.GetId()),
			Data:        m.GetData(),
		})
	}

	return modules
}

func (v *v1Alpha2Adapter) GetName() string {
	return v.record.GetName()
}

func (v *v1Alpha2Adapter) GetPreviousRecordCid() string {
	return v.record.GetPreviousRecordCid()
}

func (v *v1Alpha2Adapter) GetSchemaVersion() string {
	return v.record.GetSchemaVersion()
}

func (v *v1Alpha2Adapter) GetSkills() []coretypes.Skill {
	skills := make([]coretypes.Skill, 0, len(v.record.GetSkills()))
	for _, s := range v.record.GetSkills() {
		skills = append(skills, &skill{
			Annotations: make(map[string]string), // v1alpha2 does not have annotations for skills
			Name:        s.GetName(),
			ID:          uint64(s.GetId()),
		})
	}

	return skills
}

func (v *v1Alpha2Adapter) GetAnnotations() map[string]string {
	return v.record.GetAnnotations()
}

func (v *v1Alpha2Adapter) GetAuthors() []string {
	return v.record.GetAuthors()
}

func (v *v1Alpha2Adapter) GetCreatedAt() string {
	return v.record.GetCreatedAt()
}

func (v *v1Alpha2Adapter) GetDescription() string {
	return v.record.GetDescription()
}

func (v *v1Alpha2Adapter) GetDomains() []coretypes.Domain {
	domains := make([]coretypes.Domain, 0, len(v.record.GetDomains()))
	for _, d := range v.record.GetDomains() {
		domains = append(domains, &domain{
			Annotations: make(map[string]string), // v1alpha2 does not have annotations for domains
			Name:        d.GetName(),
			ID:          uint64(d.GetId()),
		})
	}

	return domains
}

func (v *v1Alpha2Adapter) GetLocators() []coretypes.Locator {
	locators := make([]coretypes.Locator, 0, len(v.record.GetLocators()))
	for _, l := range v.record.GetLocators() {
		locators = append(locators, &locator{
			Annotations: l.GetAnnotations(),
			Type:        l.GetType(),
			URL:         l.GetUrl(),
		})
	}

	return locators
}

func (v *v1Alpha2Adapter) GetVersion() string {
	return v.record.GetVersion()
}
