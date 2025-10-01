// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// V1Alpha1Adapter adapts typesv1alpha1.Record to types.RecordData interface.
type V1Alpha1Adapter struct {
	record *typesv1alpha1.Record
}

// Compile-time interface checks.
var (
	_ types.RecordData    = (*V1Alpha1Adapter)(nil)
	_ types.LabelProvider = (*V1Alpha1Adapter)(nil)
)

// NewV1Alpha1Adapter creates a new V1Alpha1Adapter.
func NewV1Alpha1Adapter(record *typesv1alpha1.Record) *V1Alpha1Adapter {
	return &V1Alpha1Adapter{record: record}
}

// GetAnnotations implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetAnnotations() map[string]string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAnnotations()
}

// GetSchemaVersion implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetSchemaVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetSchemaVersion()
}

// GetName implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetName() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetName()
}

// GetVersion implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetVersion()
}

// GetDescription implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetDescription() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetDescription()
}

// GetAuthors implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetAuthors() []string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetCreatedAt() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetCreatedAt()
}

// GetSkills implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetSkills() []types.Skill {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()
	result := make([]types.Skill, len(skills))

	for i, skill := range skills {
		result[i] = NewV1Alpha1SkillAdapter(skill)
	}

	return result
}

// GetDomains implements types.RecordData.
func (a *V1Alpha1Adapter) GetDomains() []types.Domain {
	if a.record == nil {
		return nil
	}

	domains := a.record.GetDomains()
	result := make([]types.Domain, len(domains))

	for i, domain := range domains {
		result[i] = NewV1Alpha1DomainAdapter(domain)
	}

	return result
}

// GetLocators implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetLocators() []types.Locator {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Locator, len(locators))

	for i, locator := range locators {
		result[i] = NewV1Alpha1LocatorAdapter(locator)
	}

	return result
}

// GetModules implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetModules() []types.Module {
	if a.record == nil {
		return nil
	}

	modules := a.record.GetModules()
	result := make([]types.Module, len(modules))

	for i, module := range modules {
		result[i] = NewV1Alpha1ModuleAdapter(module)
	}

	return result
}

// GetSignature implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetSignature() types.Signature {
	if a.record == nil || a.record.GetSignature() == nil {
		return nil
	}

	return NewV1Alpha1SignatureAdapter(a.record.GetSignature())
}

// GetPreviousRecordCid implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetPreviousRecordCid() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetPreviousRecordCid()
}

// V1Alpha1SignatureAdapter adapts typesv1alpha1.Signature to types.Signature interface.
type V1Alpha1SignatureAdapter struct {
	signature *typesv1alpha1.Signature
}

// NewV1Alpha1SignatureAdapter creates a new V1Alpha1SignatureAdapter.
func NewV1Alpha1SignatureAdapter(signature *typesv1alpha1.Signature) *V1Alpha1SignatureAdapter {
	return &V1Alpha1SignatureAdapter{signature: signature}
}

// GetAnnotations implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetAnnotations() map[string]string {
	if s.signature == nil {
		return nil
	}

	return s.signature.GetAnnotations()
}

// GetSignedAt implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetSignedAt() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetSignedAt()
}

// GetAlgorithm implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetAlgorithm() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetAlgorithm()
}

// GetSignature implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetSignature() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetSignature()
}

// GetCertificate implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetCertificate() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetCertificate()
}

// GetContentType implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetContentType() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetContentType()
}

// GetContentBundle implements types.Signature interface.
func (s *V1Alpha1SignatureAdapter) GetContentBundle() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetContentBundle()
}

// V1Alpha1SkillAdapter adapts typesv1alpha1.Skill to types.Skill interface.
type V1Alpha1SkillAdapter struct {
	skill *typesv1alpha1.Skill
}

// NewV1Alpha1SkillAdapter creates a new V1Alpha1SkillAdapter.
func NewV1Alpha1SkillAdapter(skill *typesv1alpha1.Skill) *V1Alpha1SkillAdapter {
	return &V1Alpha1SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface.
func (s *V1Alpha1SkillAdapter) GetAnnotations() map[string]string {
	if s.skill == nil {
		return nil
	}

	return s.skill.GetAnnotations()
}

// GetName implements types.Skill interface.
func (s *V1Alpha1SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}

	return s.skill.GetName()
}

// GetID implements types.Skill interface.
func (s *V1Alpha1SkillAdapter) GetID() uint64 {
	if s.skill == nil {
		return 0
	}

	return uint64(s.skill.GetId())
}

// V1Alpha1LocatorAdapter adapts typesv1alpha1.Locator to types.Locator interface.
type V1Alpha1LocatorAdapter struct {
	locator *typesv1alpha1.Locator
}

// NewV1Alpha1LocatorAdapter creates a new V1Alpha1LocatorAdapter.
func NewV1Alpha1LocatorAdapter(locator *typesv1alpha1.Locator) *V1Alpha1LocatorAdapter {
	return &V1Alpha1LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface.
func (l *V1Alpha1LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}

	return l.locator.GetAnnotations()
}

// GetType implements types.Locator interface.
func (l *V1Alpha1LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetType()
}

// GetURL implements types.Locator interface.
func (l *V1Alpha1LocatorAdapter) GetURL() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetUrl()
}

// GetSize implements types.Locator interface.
func (l *V1Alpha1LocatorAdapter) GetSize() uint64 {
	if l.locator == nil {
		return 0
	}

	return l.locator.GetSize()
}

// GetDigest implements types.Locator interface.
func (l *V1Alpha1LocatorAdapter) GetDigest() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetDigest()
}

// V1Alpha1SkillAdapter adapts typesv1alpha1.Skill to types.Skill interface.
type V1Alpha1DomainAdapter struct {
	domain *typesv1alpha1.Domain
}

// NewV1Alpha1DomainAdapter creates a new V1Alpha1DomainAdapter.
func NewV1Alpha1DomainAdapter(domain *typesv1alpha1.Domain) *V1Alpha1DomainAdapter {
	if domain == nil {
		return nil
	}

	return &V1Alpha1DomainAdapter{domain: domain}
}

// GetAnnotations implements types.Domain interface.
func (d *V1Alpha1DomainAdapter) GetAnnotations() map[string]string {
	if d.domain == nil {
		return nil
	}

	return d.domain.GetAnnotations()
}

// GetName implements types.Domain interface.
func (d *V1Alpha1DomainAdapter) GetName() string {
	if d.domain == nil {
		return ""
	}

	return d.domain.GetName()
}

// GetID implements types.Domain interface.
func (d *V1Alpha1DomainAdapter) GetID() uint64 {
	if d.domain == nil {
		return 0
	}

	return uint64(d.domain.GetId())
}

// V1Alpha1ModuleAdapter adapts typesv1alpha1.Module to types.Module interface.
type V1Alpha1ModuleAdapter struct {
	module *typesv1alpha1.Module
}

// NewV1Alpha1ModuleAdapter creates a new V1Alpha1ModuleAdapter.
func NewV1Alpha1ModuleAdapter(module *typesv1alpha1.Module) *V1Alpha1ModuleAdapter {
	return &V1Alpha1ModuleAdapter{module: module}
}

// GetName implements types.Module interface.
func (m *V1Alpha1ModuleAdapter) GetName() string {
	if m.module == nil {
		return ""
	}

	return m.module.GetName()
}

// GetData implements types.Module interface.
func (m *V1Alpha1ModuleAdapter) GetData() map[string]any {
	if m.module == nil || m.module.GetData() == nil {
		return nil
	}

	resp, err := decoder.ProtoToStruct[map[string]any](m.module.GetData())
	if err != nil {
		return nil
	}

	return *resp
}

// GetSkillLabels implements types.LabelProvider interface.
func (a *V1Alpha1Adapter) GetSkillLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()
	result := make([]types.Label, 0, len(skills))

	for _, skill := range skills {
		// Reuse the existing skill adapter logic for name formatting
		skillAdapter := NewV1Alpha1SkillAdapter(skill)
		skillName := skillAdapter.GetName()

		skillLabel := types.Label(types.LabelTypeSkill.Prefix() + skillName)
		result = append(result, skillLabel)
	}

	return result
}

// GetLocatorLabels implements types.LabelProvider interface.
func (a *V1Alpha1Adapter) GetLocatorLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Label, 0, len(locators))

	for _, locator := range locators {
		locatorAdapter := NewV1Alpha1LocatorAdapter(locator)
		locatorType := locatorAdapter.GetType()

		locatorLabel := types.Label(types.LabelTypeLocator.Prefix() + locatorType)
		result = append(result, locatorLabel)
	}

	return result
}

// GetDomainLabels implements types.LabelProvider interface.
func (a *V1Alpha1Adapter) GetDomainLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	domains := a.record.GetDomains()
	result := make([]types.Label, 0, len(domains))

	for _, domain := range domains {
		domainAdapter := NewV1Alpha1DomainAdapter(domain)
		domainName := domainAdapter.GetName()

		domainLabel := types.Label(types.LabelTypeDomain.Prefix() + domainName)
		result = append(result, domainLabel)
	}

	return result
}

// GetModuleLabels implements types.LabelProvider interface.
func (a *V1Alpha1Adapter) GetModuleLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	modules := a.record.GetModules()
	result := make([]types.Label, 0, len(modules))

	for _, mod := range modules {
		moduleAdapter := NewV1Alpha1ModuleAdapter(mod)
		moduleName := moduleAdapter.GetName()

		// V1Alpha1 modules don't have schema prefixes, use name directly with /modules prefix
		moduleLabel := types.Label(types.LabelTypeModule.Prefix() + moduleName)
		result = append(result, moduleLabel)
	}

	return result
}

// GetAllLabels implements types.LabelProvider interface.
func (a *V1Alpha1Adapter) GetAllLabels() []types.Label {
	var allLabels []types.Label

	allLabels = append(allLabels, a.GetSkillLabels()...)
	allLabels = append(allLabels, a.GetDomainLabels()...)
	allLabels = append(allLabels, a.GetModuleLabels()...)
	allLabels = append(allLabels, a.GetLocatorLabels()...)

	return allLabels
}
