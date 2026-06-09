// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	coretypes "github.com/agntcy/dir/api/core/types"
)

type v1Alpha1Adapter struct {
	cid    string
	record *typesv1alpha1.Record
}

func newV1Alpha1Adapter(cid string, record *typesv1alpha1.Record) coretypes.Record {
	return &v1Alpha1Adapter{
		cid:    cid,
		record: record,
	}
}

func (v *v1Alpha1Adapter) GetCid() string {
	return v.cid
}

func (v *v1Alpha1Adapter) GetModules() []coretypes.Module {
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

func (v *v1Alpha1Adapter) GetName() string {
	return v.record.GetName()
}

func (v *v1Alpha1Adapter) GetPreviousRecordCid() string {
	return v.record.GetPreviousRecordCid()
}

func (v *v1Alpha1Adapter) GetSchemaVersion() string {
	return v.record.GetSchemaVersion()
}

func (v *v1Alpha1Adapter) GetSkills() []coretypes.Skill {
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

func (v *v1Alpha1Adapter) GetAnnotations() map[string]string {
	return v.record.GetAnnotations()
}

func (v *v1Alpha1Adapter) GetAuthors() []string {
	return v.record.GetAuthors()
}

func (v *v1Alpha1Adapter) GetCreatedAt() string {
	return v.record.GetCreatedAt()
}

func (v *v1Alpha1Adapter) GetDescription() string {
	return v.record.GetDescription()
}

func (v *v1Alpha1Adapter) GetDomains() []coretypes.Domain {
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

func (v *v1Alpha1Adapter) GetLocators() []coretypes.Locator {
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

func (v *v1Alpha1Adapter) GetVersion() string {
	return v.record.GetVersion()
}
