// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"
	"strings"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha0"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

const featuresSchemaPrefix = "schema.oasf.agntcy.org/features/"

// V1Alpha0Adapter adapts typesv1alpha0.Record to types.RecordData interface.
type V1Alpha0Adapter struct {
	record *typesv1alpha0.Record
}

// Compile-time interface checks.
var (
	_ types.RecordData    = (*V1Alpha0Adapter)(nil)
	_ types.LabelProvider = (*V1Alpha0Adapter)(nil)
)

// NewV1Alpha0Adapter creates a new V1Alpha0Adapter.
func NewV1Alpha0Adapter(record *typesv1alpha0.Record) *V1Alpha0Adapter {
	return &V1Alpha0Adapter{record: record}
}

// GetAnnotations implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetAnnotations() map[string]string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAnnotations()
}

// GetSchemaVersion implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetSchemaVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetSchemaVersion()
}

// GetDomains implements types.RecordData.
//
// NOTE: V1Alpha0 doesn't have domains, so we return an empty slice.
func (a *V1Alpha0Adapter) GetDomains() []types.Domain {
	return []types.Domain{}
}

// GetName implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetName() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetName()
}

// GetVersion implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetVersion()
}

// GetDescription implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetDescription() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetDescription()
}

// GetAuthors implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetAuthors() []string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetCreatedAt() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetCreatedAt()
}

// GetSkills implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetSkills() []types.Skill {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()
	result := make([]types.Skill, len(skills))

	for i, skill := range skills {
		result[i] = NewV1Alpha0SkillAdapter(skill)
	}

	return result
}

// GetLocators implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetLocators() []types.Locator {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Locator, len(locators))

	for i, locator := range locators {
		result[i] = NewV1Alpha0LocatorAdapter(locator)
	}

	return result
}

// GetModules implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetModules() []types.Module {
	if a.record == nil {
		return nil
	}

	extensions := a.record.GetExtensions()
	result := make([]types.Module, len(extensions))

	for i, extension := range extensions {
		result[i] = NewV1Alpha0ModuleAdapter(extension)
	}

	return result
}

// GetSignature implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetSignature() types.Signature {
	if a.record == nil || a.record.GetSignature() == nil {
		return nil
	}

	return NewV1Alpha0SignatureAdapter(a.record.GetSignature())
}

// GetPreviousRecordCid implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetPreviousRecordCid() string {
	// V1 doesn't have previous record CID
	return ""
}

// V1Alpha0SignatureAdapter adapts typesv1alpha0.Signature to types.Signature interface.
type V1Alpha0SignatureAdapter struct {
	signature *typesv1alpha0.Signature
}

// NewV1Alpha0SignatureAdapter creates a new V1Alpha0SignatureAdapter.
func NewV1Alpha0SignatureAdapter(signature *typesv1alpha0.Signature) *V1Alpha0SignatureAdapter {
	return &V1Alpha0SignatureAdapter{signature: signature}
}

// GetAnnotations implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetAnnotations() map[string]string {
	// V1 signature doesn't have annotations
	return nil
}

// GetSignedAt implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetSignedAt() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetSignedAt()
}

// GetAlgorithm implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetAlgorithm() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetAlgorithm()
}

// GetSignature implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetSignature() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetSignature()
}

// GetCertificate implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetCertificate() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetCertificate()
}

// GetContentType implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetContentType() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetContentType()
}

// GetContentBundle implements types.Signature interface.
func (s *V1Alpha0SignatureAdapter) GetContentBundle() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetContentBundle()
}

// V1Alpha0SkillAdapter adapts typesv1alpha0.Skill to types.Skill interface.
type V1Alpha0SkillAdapter struct {
	skill *typesv1alpha0.Skill
}

// NewV1Alpha0SkillAdapter creates a new V1Alpha0SkillAdapter.
func NewV1Alpha0SkillAdapter(skill *typesv1alpha0.Skill) *V1Alpha0SkillAdapter {
	return &V1Alpha0SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface.
func (s *V1Alpha0SkillAdapter) GetAnnotations() map[string]string {
	if s.skill == nil {
		return nil
	}

	return s.skill.GetAnnotations()
}

// GetName implements types.Skill interface.
func (s *V1Alpha0SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}

	if s.skill.GetClassName() == "" {
		return s.skill.GetCategoryName()
	}

	return fmt.Sprintf("%s/%s", s.skill.GetCategoryName(), s.skill.GetClassName())
}

// GetID implements types.Skill interface.
func (s *V1Alpha0SkillAdapter) GetID() uint64 {
	if s.skill == nil {
		return 0
	}

	return s.skill.GetClassUid()
}

// V1Alpha0LocatorAdapter adapts typesv1alpha0.Locator to types.Locator interface.
type V1Alpha0LocatorAdapter struct {
	locator *typesv1alpha0.Locator
}

// NewV1Alpha0LocatorAdapter creates a new V1Alpha0LocatorAdapter.
func NewV1Alpha0LocatorAdapter(locator *typesv1alpha0.Locator) *V1Alpha0LocatorAdapter {
	return &V1Alpha0LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface.
func (l *V1Alpha0LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}

	return l.locator.GetAnnotations()
}

// GetType implements types.Locator interface.
func (l *V1Alpha0LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetType()
}

// GetURL implements types.Locator interface.
func (l *V1Alpha0LocatorAdapter) GetURL() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetUrl()
}

// GetSize implements types.Locator interface.
func (l *V1Alpha0LocatorAdapter) GetSize() uint64 {
	if l.locator == nil {
		return 0
	}

	return l.locator.GetSize()
}

// GetDigest implements types.Locator interface.
func (l *V1Alpha0LocatorAdapter) GetDigest() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetDigest()
}

// V1Alpha0ModuleAdapter adapts typesv1alpha0.Extension to types.Module interface.
type V1Alpha0ModuleAdapter struct {
	extension *typesv1alpha0.Extension
}

// NewV1Alpha0ModuleAdapter creates a new V1Alpha0ModuleAdapter.
func NewV1Alpha0ModuleAdapter(extension *typesv1alpha0.Extension) *V1Alpha0ModuleAdapter {
	return &V1Alpha0ModuleAdapter{extension: extension}
}

// GetName implements types.Module interface.
func (m *V1Alpha0ModuleAdapter) GetName() string {
	if m.extension == nil {
		return ""
	}

	return m.extension.GetName()
}

// GetData implements types.Module interface.
func (m *V1Alpha0ModuleAdapter) GetData() map[string]any {
	if m.extension == nil || m.extension.GetData() == nil {
		return nil
	}

	resp, err := decoder.ProtoToStruct[map[string]any](m.extension.GetData())
	if err != nil {
		return nil
	}

	return *resp
}

// GetSkillLabels implements types.LabelProvider interface.
func (a *V1Alpha0Adapter) GetSkillLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()
	result := make([]types.Label, 0, len(skills))

	for _, skill := range skills {
		// Reuse the existing skill adapter logic for name formatting
		skillAdapter := NewV1Alpha0SkillAdapter(skill)
		skillName := skillAdapter.GetName()

		skillLabel := types.Label(types.LabelTypeSkill.Prefix() + skillName)
		result = append(result, skillLabel)
	}

	return result
}

// GetLocatorLabels implements types.LabelProvider interface.
func (a *V1Alpha0Adapter) GetLocatorLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Label, 0, len(locators))

	for _, locator := range locators {
		locatorAdapter := NewV1Alpha0LocatorAdapter(locator)
		locatorType := locatorAdapter.GetType()

		locatorLabel := types.Label(types.LabelTypeLocator.Prefix() + locatorType)
		result = append(result, locatorLabel)
	}

	return result
}

// GetDomainLabels implements types.LabelProvider interface.
func (a *V1Alpha0Adapter) GetDomainLabels() []types.Label {
	// V1Alpha0 doesn't have domains, return empty slice
	return []types.Label{}
}

// GetModuleLabels implements types.LabelProvider interface.
func (a *V1Alpha0Adapter) GetModuleLabels() []types.Label {
	if a.record == nil {
		return nil
	}

	extensions := a.record.GetExtensions()
	result := make([]types.Label, 0, len(extensions))

	for _, ext := range extensions {
		extensionName := ext.GetName()

		// Handle v0.3.1 schema prefix for features - now map to modules
		name := strings.TrimPrefix(extensionName, featuresSchemaPrefix)
		moduleLabel := types.Label(types.LabelTypeModule.Prefix() + name)
		result = append(result, moduleLabel)
	}

	return result
}

// GetAllLabels implements types.LabelProvider interface.
func (a *V1Alpha0Adapter) GetAllLabels() []types.Label {
	var allLabels []types.Label

	allLabels = append(allLabels, a.GetSkillLabels()...)
	allLabels = append(allLabels, a.GetDomainLabels()...)
	allLabels = append(allLabels, a.GetModuleLabels()...)
	allLabels = append(allLabels, a.GetLocatorLabels()...)

	return allLabels
}
