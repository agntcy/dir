// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	coretypes "github.com/agntcy/dir/api/core/types"
)

type v1Adapter struct {
	record *typesv1.Record
}

func newV1Adapter(record *typesv1.Record) coretypes.RecordReader {
	return &v1Adapter{
		record: record,
	}
}

// GetAnnotations implements [Record].
func (v *v1Adapter) GetAnnotations() map[string]string {
	return v.record.GetAnnotations()
}

// GetAuthors implements [Record].
func (v *v1Adapter) GetAuthors() []string {
	return v.record.GetAuthors()
}

// GetCreatedAt implements [Record].
func (v *v1Adapter) GetCreatedAt() string {
	return v.record.GetCreatedAt()
}

// GetDescription implements [Record].
func (v *v1Adapter) GetDescription() string {
	return v.record.GetDescription()
}

// GetDomains implements [Record].
func (v *v1Adapter) GetDomains() []coretypes.Domain {
	domains := make([]coretypes.Domain, 0, len(v.record.GetDomains()))
	for _, d := range v.record.GetDomains() {
		domains = append(domains, coretypes.Domain{
			Annotations: d.GetAnnotations(),
			Name:        d.GetName(),
			ID:          uint64(d.GetId()),
		})
	}
	return domains
}

// GetLocators implements [Record].
func (v *v1Adapter) GetLocators() []coretypes.Locator {
	locators := make([]coretypes.Locator, 0, len(v.record.GetLocators()))
	for _, l := range v.record.GetLocators() {
		locators = append(locators, coretypes.Locator{
			Annotations: l.GetAnnotations(),
			Type:        l.GetType(),
			URLs:        l.GetUrls(),
		})
	}
	return locators
}

// GetModules implements [Record].
func (v *v1Adapter) GetModules() []coretypes.Module {
	modules := make([]coretypes.Module, 0, len(v.record.GetModules()))
	for _, m := range v.record.GetModules() {
		modules = append(modules, coretypes.Module{
			Annotations: m.GetAnnotations(),
			Name:        m.GetName(),
			ID:          uint64(m.GetId()),
			Data:        m.GetData(),
		})
	}
	return modules
}

// GetName implements [Record].
func (v *v1Adapter) GetName() string {
	return v.record.GetName()
}

// GetPreviousRecordCid implements [Record].
func (v *v1Adapter) GetPreviousRecordCid() string {
	return ""
}

// GetSchemaVersion implements [Record].
func (v *v1Adapter) GetSchemaVersion() string {
	return v.record.GetSchemaVersion()
}

// GetSkills implements [Record].
func (v *v1Adapter) GetSkills() []coretypes.Skill {
	skills := make([]coretypes.Skill, 0, len(v.record.GetSkills()))
	for _, s := range v.record.GetSkills() {
		skills = append(skills, coretypes.Skill{
			Annotations: s.GetAnnotations(),
			Name:        s.GetName(),
			ID:          uint64(s.GetId()),
		})
	}
	return skills
}

// GetVersion implements [Record].
func (v *v1Adapter) GetVersion() string {
	return v.record.GetVersion()
}
