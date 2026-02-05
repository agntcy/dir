// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// V1Adapter adapts typesv1.Record to types.RecordData interface.
// This adapter is for OASF 1.0.0 schema.
type V1Adapter struct {
	record *typesv1.Record
}

// Compile-time interface checks.
var (
	_ types.RecordData    = (*V1Adapter)(nil)
	_ types.LabelProvider = (*V1Adapter)(nil)
)

// NewV1Adapter creates a new V1Adapter.
func NewV1Adapter(record *typesv1.Record) *V1Adapter {
	return &V1Adapter{record: record}
}

// GetAnnotations implements types.RecordData interface.
func (a *V1Adapter) GetAnnotations() map[string]string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAnnotations()
}

// GetAuthors implements types.RecordData interface.
func (a *V1Adapter) GetAuthors() []string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface.
func (a *V1Adapter) GetCreatedAt() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetCreatedAt()
}

// GetDescription implements types.RecordData interface.
func (a *V1Adapter) GetDescription() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetDescription()
}

// GetDomains implements types.RecordData interface.
func (a *V1Adapter) GetDomains() []types.Domain {
	if a.record == nil {
		return nil
	}

	domains := a.record.GetDomains()
	result := make([]types.Domain, len(domains))

	for i, domain := range domains {
		result[i] = NewV1DomainAdapter(domain)
	}

	return result
}

// V1DomainAdapter adapts typesv1.Domain to types.Domain interface.
type V1DomainAdapter struct {
	domain *typesv1.Domain
}

// NewV1DomainAdapter creates a new V1DomainAdapter.
func NewV1DomainAdapter(domain *typesv1.Domain) *V1DomainAdapter {
	if domain == nil {
		return nil
	}

	return &V1DomainAdapter{domain: domain}
}

// GetAnnotations implements types.Domain interface.
func (d *V1DomainAdapter) GetAnnotations() map[string]string {
	return nil
}

// GetID implements types.Domain interface.
func (d *V1DomainAdapter) GetID() uint64 {
	if d.domain == nil {
		return 0
	}

	return uint64(d.domain.GetId())
}

// GetName implements types.Domain interface.
func (d *V1DomainAdapter) GetName() string {
	if d.domain == nil {
		return ""
	}

	return d.domain.GetName()
}

// GetLocators implements types.RecordData interface.
func (a *V1Adapter) GetLocators() []types.Locator {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Locator, len(locators))

	for i, locator := range locators {
		result[i] = NewV1LocatorAdapter(locator)
	}

	return result
}

// V1LocatorAdapter adapts typesv1.Locator to types.Locator interface.
type V1LocatorAdapter struct {
	locator *typesv1.Locator
}

// NewV1LocatorAdapter creates a new V1LocatorAdapter.
func NewV1LocatorAdapter(locator *typesv1.Locator) *V1LocatorAdapter {
	if locator == nil {
		return nil
	}

	return &V1LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface.
func (l *V1LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}

	return l.locator.GetAnnotations()
}

// GetDigest implements types.Locator interface.
// Note: v1 Locator does not have digest field.
func (l *V1LocatorAdapter) GetDigest() string {
	return ""
}

// GetSize implements types.Locator interface.
// Note: v1 Locator does not have size field.
func (l *V1LocatorAdapter) GetSize() uint64 {
	return 0
}

// GetType implements types.Locator interface.
func (l *V1LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetType()
}

// GetURL implements types.Locator interface.
// For v1 (1.0.0), locators use urls (plural) instead of url (singular).
func (l *V1LocatorAdapter) GetURL() string {
	if l.locator == nil {
		return ""
	}

	// v1 uses GetUrls() which returns a slice
	urls := l.locator.GetUrls()
	if len(urls) > 0 {
		return urls[0]
	}

	return ""
}

// GetModules implements types.RecordData interface.
func (a *V1Adapter) GetModules() []types.Module {
	if a.record == nil {
		return nil
	}

	modules := a.record.GetModules()

	result := make([]types.Module, len(modules))
	for i, module := range modules {
		result[i] = NewV1ModuleAdapter(module)
	}

	return result
}

// V1ModuleAdapter adapts typesv1.Module to types.Module interface.
type V1ModuleAdapter struct {
	module *typesv1.Module
}

// NewV1ModuleAdapter creates a new V1ModuleAdapter.
func NewV1ModuleAdapter(module *typesv1.Module) *V1ModuleAdapter {
	if module == nil {
		return nil
	}

	return &V1ModuleAdapter{module: module}
}

// GetData implements types.Module interface.
func (m *V1ModuleAdapter) GetData() map[string]any {
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
func (m *V1ModuleAdapter) GetName() string {
	if m.module == nil {
		return ""
	}

	return m.module.GetName()
}

// GetID implements types.Module interface.
func (m *V1ModuleAdapter) GetID() uint64 {
	if m.module == nil {
		return 0
	}

	return uint64(m.module.GetId())
}

// GetName implements types.RecordData interface.
func (a *V1Adapter) GetName() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetName()
}

// GetPreviousRecordCid implements types.RecordData interface.
// Note: v1 Record does not have previous_record_cid field.
func (a *V1Adapter) GetPreviousRecordCid() string {
	return ""
}

// GetSchemaVersion implements types.RecordData interface.
func (a *V1Adapter) GetSchemaVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetSchemaVersion()
}

// GetSignature implements types.RecordData interface.
func (a *V1Adapter) GetSignature() types.Signature {
	return nil
}

// GetSkills implements types.RecordData interface.
func (a *V1Adapter) GetSkills() []types.Skill {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()

	result := make([]types.Skill, len(skills))
	for i, skill := range skills {
		result[i] = NewV1SkillAdapter(skill)
	}

	return result
}

// V1SkillAdapter adapts typesv1.Skill to types.Skill interface.
type V1SkillAdapter struct {
	skill *typesv1.Skill
}

// NewV1SkillAdapter creates a new V1SkillAdapter.
func NewV1SkillAdapter(skill *typesv1.Skill) *V1SkillAdapter {
	if skill == nil {
		return nil
	}

	return &V1SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface.
func (s *V1SkillAdapter) GetAnnotations() map[string]string {
	return nil
}

// GetID implements types.Skill interface.
func (s *V1SkillAdapter) GetID() uint64 {
	if s.skill == nil {
		return 0
	}

	return uint64(s.skill.GetId())
}

// GetName implements types.Skill interface.
func (s *V1SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}

	return s.skill.GetName()
}

// GetVersion implements types.RecordData interface.
func (a *V1Adapter) GetVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetVersion()
}

// GetDomainLabels implements types.LabelProvider interface.
func (a *V1Adapter) GetDomainLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	domains := a.record.GetDomains()
	result := make([]types.Label, 0, len(domains))

	for _, domain := range domains {
		domainAdapter := NewV1DomainAdapter(domain)
		domainName := domainAdapter.GetName()

		domainLabel := types.Label(types.LabelTypeDomain.Prefix() + domainName)
		result = append(result, domainLabel)
	}

	return result
}

// GetLocatorLabels implements types.LabelProvider interface.
func (a *V1Adapter) GetLocatorLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Label, 0, len(locators))

	for _, locator := range locators {
		locatorAdapter := NewV1LocatorAdapter(locator)
		locatorType := locatorAdapter.GetType()

		locatorLabel := types.Label(types.LabelTypeLocator.Prefix() + locatorType)
		result = append(result, locatorLabel)
	}

	return result
}

// GetModuleLabels implements types.LabelProvider interface.
func (a *V1Adapter) GetModuleLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	modules := a.record.GetModules()
	result := make([]types.Label, 0, len(modules))

	for _, module := range modules {
		moduleAdapter := NewV1ModuleAdapter(module)
		moduleName := moduleAdapter.GetName()

		moduleLabel := types.Label(types.LabelTypeModule.Prefix() + moduleName)
		result = append(result, moduleLabel)
	}

	return result
}

// GetSkillLabels implements types.LabelProvider interface.
func (a *V1Adapter) GetSkillLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()

	result := make([]types.Label, 0, len(skills))
	for _, skill := range skills {
		skillAdapter := NewV1SkillAdapter(skill)
		skillName := skillAdapter.GetName()

		skillLabel := types.Label(types.LabelTypeSkill.Prefix() + skillName)
		result = append(result, skillLabel)
	}

	return result
}

// GetAllLabels implements types.LabelProvider interface.
func (a *V1Adapter) GetAllLabels() []types.Label {
	var allLabels []types.Label

	allLabels = append(allLabels, a.GetDomainLabels()...)
	allLabels = append(allLabels, a.GetLocatorLabels()...)
	allLabels = append(allLabels, a.GetModuleLabels()...)
	allLabels = append(allLabels, a.GetSkillLabels()...)

	return allLabels
}
