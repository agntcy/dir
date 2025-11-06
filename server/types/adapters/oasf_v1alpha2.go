// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1alpha2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// V1Alpha2Adapter adapts typesv1alpha2.Record to types.RecordData interface.
type V1Alpha2Adapter struct {
	record *typesv1alpha2.Record
}

// Compile-time interface checks.
var (
	_ types.RecordData    = (*V1Alpha2Adapter)(nil)
	_ types.LabelProvider = (*V1Alpha2Adapter)(nil)
)

// NewV1Alpha2Adapter creates a new V1Alpha2Adapter.
func NewV1Alpha2Adapter(record *typesv1alpha2.Record) *V1Alpha2Adapter {
	return &V1Alpha2Adapter{record: record}
}

// GetAnnotations implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetAnnotations() map[string]string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAnnotations()
}

// GetAuthors implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetAuthors() []string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetCreatedAt() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetCreatedAt()
}

// GetDescription implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetDescription() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetDescription()
}

// GetDomains implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetDomains() []types.Domain {
	if a.record == nil {
		return nil
	}

	domains := a.record.GetDomains()
	result := make([]types.Domain, len(domains))

	for i, domain := range domains {
		result[i] = NewV1Alpha2DomainAdapter(domain)
	}

	return result
}

// V1Alpha2DomainAdapter adapts typesv1alpha2.Domain to types.Domain interface.
type V1Alpha2DomainAdapter struct {
	domain *typesv1alpha2.Domain
}

// NewV1Alpha2DomainAdapter creates a new V1Alpha2DomainAdapter.
func NewV1Alpha2DomainAdapter(domain *typesv1alpha2.Domain) *V1Alpha2DomainAdapter {
	if domain == nil {
		return nil
	}

	return &V1Alpha2DomainAdapter{domain: domain}
}

// GetAnnotations implements types.Domain interface.
func (d *V1Alpha2DomainAdapter) GetAnnotations() map[string]string {
	return nil
}

// GetID implements types.Domain interface.
func (d *V1Alpha2DomainAdapter) GetID() uint64 {
	if d.domain == nil {
		return 0
	}

	return uint64(d.domain.GetId())
}

// GetName implements types.Domain interface.
func (d *V1Alpha2DomainAdapter) GetName() string {
	if d.domain == nil {
		return ""
	}

	return d.domain.GetName()
}

// GetLocators implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetLocators() []types.Locator {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Locator, len(locators))

	for i, locator := range locators {
		result[i] = NewV1Alpha2LocatorAdapter(locator)
	}

	return result
}

// V1Alpha2LocatorAdapter adapts typesv1alpha2.Locator to types.Locator interface.
type V1Alpha2LocatorAdapter struct {
	locator *typesv1alpha2.Locator
}

// NewV1Alpha2LocatorAdapter creates a new V1Alpha2LocatorAdapter.
func NewV1Alpha2LocatorAdapter(locator *typesv1alpha2.Locator) *V1Alpha2LocatorAdapter {
	if locator == nil {
		return nil
	}

	return &V1Alpha2LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface.
func (l *V1Alpha2LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}

	return l.locator.GetAnnotations()
}

// GetDigest implements types.Locator interface.
func (l *V1Alpha2LocatorAdapter) GetDigest() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetDigest()
}

// GetSize implements types.Locator interface.
func (l *V1Alpha2LocatorAdapter) GetSize() uint64 {
	if l.locator == nil {
		return 0
	}

	return l.locator.GetSize()
}

// GetType implements types.Locator interface.
func (l *V1Alpha2LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetType()
}

// GetURL implements types.Locator interface.
func (l *V1Alpha2LocatorAdapter) GetURL() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetUrl()
}

// GetModules implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetModules() []types.Module {
	if a.record == nil {
		return nil
	}

	modules := a.record.GetModules()

	result := make([]types.Module, len(modules))
	for i, module := range modules {
		result[i] = NewV1Alpha2ModuleAdapter(module)
	}

	return result
}

// V1Alpha2ModuleAdapter adapts typesv1alpha2.Module to types.Module interface.
type V1Alpha2ModuleAdapter struct {
	module *typesv1alpha2.Module
}

// NewV1Alpha2ModuleAdapter creates a new V1Alpha2ModuleAdapter.
func NewV1Alpha2ModuleAdapter(module *typesv1alpha2.Module) *V1Alpha2ModuleAdapter {
	if module == nil {
		return nil
	}

	return &V1Alpha2ModuleAdapter{module: module}
}

// GetData implements types.Module interface.
func (m *V1Alpha2ModuleAdapter) GetData() map[string]any {
	if m.module == nil || m.module.GetData() == nil {
		return nil
	}

	resp, err := decoder.ProtoToStruct[map[string]any](m.module.GetData())
	if err != nil {
		return nil
	}

	return *resp
}

// GetName implements types.Module interface.
func (m *V1Alpha2ModuleAdapter) GetName() string {
	if m.module == nil {
		return ""
	}

	return m.module.GetName()
}

// GetName implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetName() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetName()
}

// GetPreviousRecordCid implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetPreviousRecordCid() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetPreviousRecordCid()
}

// GetSchemaVersion implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetSchemaVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetSchemaVersion()
}

// GetSignature implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetSignature() types.Signature {
	return nil
}

// GetSkills implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetSkills() []types.Skill {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()

	result := make([]types.Skill, len(skills))
	for i, skill := range skills {
		result[i] = NewV1Alpha2SkillAdapter(skill)
	}

	return result
}

// V1Alpha2SkillAdapter adapts typesv1alpha2.Skill to types.Skill interface.
type V1Alpha2SkillAdapter struct {
	skill *typesv1alpha2.Skill
}

// NewV1Alpha2SkillAdapter creates a new V1Alpha2SkillAdapter.
func NewV1Alpha2SkillAdapter(skill *typesv1alpha2.Skill) *V1Alpha2SkillAdapter {
	if skill == nil {
		return nil
	}

	return &V1Alpha2SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface.
func (s *V1Alpha2SkillAdapter) GetAnnotations() map[string]string {
	return nil
}

// GetID implements types.Skill interface.
func (s *V1Alpha2SkillAdapter) GetID() uint64 {
	if s.skill == nil {
		return 0
	}

	return uint64(s.skill.GetId())
}

// GetName implements types.Skill interface.
func (s *V1Alpha2SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}

	return s.skill.GetName()
}

// GetVersion implements types.RecordData interface.
func (a *V1Alpha2Adapter) GetVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetVersion()
}

// GetDomainLabels implements types.LabelProvider interface.
func (a *V1Alpha2Adapter) GetDomainLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	domains := a.record.GetDomains()
	result := make([]types.Label, 0, len(domains))

	for _, domain := range domains {
		domainAdapter := NewV1Alpha2DomainAdapter(domain)
		domainName := domainAdapter.GetName()

		domainLabel := types.Label(types.LabelTypeDomain.Prefix() + domainName)
		result = append(result, domainLabel)
	}

	return result
}

// GetLocatorLabels implements types.LabelProvider interface.
func (a *V1Alpha2Adapter) GetLocatorLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Label, 0, len(locators))

	for _, locator := range locators {
		locatorAdapter := NewV1Alpha2LocatorAdapter(locator)
		locatorType := locatorAdapter.GetType()

		locatorLabel := types.Label(types.LabelTypeLocator.Prefix() + locatorType)
		result = append(result, locatorLabel)
	}

	return result
}

// GetModuleLabels implements types.LabelProvider interface.
func (a *V1Alpha2Adapter) GetModuleLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	modules := a.record.GetModules()
	result := make([]types.Label, 0, len(modules))

	for _, module := range modules {
		moduleAdapter := NewV1Alpha2ModuleAdapter(module)
		moduleName := moduleAdapter.GetName()

		moduleLabel := types.Label(types.LabelTypeModule.Prefix() + moduleName)
		result = append(result, moduleLabel)
	}

	return result
}

// GetSkillLabels implements types.LabelProvider interface.
func (a *V1Alpha2Adapter) GetSkillLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()

	result := make([]types.Label, 0, len(skills))
	for _, skill := range skills {
		skillAdapter := NewV1Alpha2SkillAdapter(skill)
		skillName := skillAdapter.GetName()

		skillLabel := types.Label(types.LabelTypeSkill.Prefix() + skillName)
		result = append(result, skillLabel)
	}

	return result
}

// GetAllLabels implements types.LabelProvider interface.
func (a *V1Alpha2Adapter) GetAllLabels() []types.Label {
	var allLabels []types.Label

	allLabels = append(allLabels, a.GetDomainLabels()...)
	allLabels = append(allLabels, a.GetLocatorLabels()...)
	allLabels = append(allLabels, a.GetModuleLabels()...)
	allLabels = append(allLabels, a.GetSkillLabels()...)

	return allLabels
}
