// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	objectsv2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v2"
	"github.com/agntcy/dir/server/types"
)

// V2DataAdapter adapts objectsv2.AgentRecord to types.RecordData interface.
type V2DataAdapter struct {
	record *objectsv2.AgentRecord
}

// NewV2DataAdapter creates a new V2DataAdapter.
func NewV2DataAdapter(record *objectsv2.AgentRecord) *V2DataAdapter {
	return &V2DataAdapter{record: record}
}

// GetAnnotations implements types.RecordData interface.
func (a *V2DataAdapter) GetAnnotations() map[string]string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAnnotations()
}

// GetSchemaVersion implements types.RecordData interface.
func (a *V2DataAdapter) GetSchemaVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetSchemaVersion()
}

// GetName implements types.RecordData interface.
func (a *V2DataAdapter) GetName() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetName()
}

// GetVersion implements types.RecordData interface.
func (a *V2DataAdapter) GetVersion() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetVersion()
}

// GetDescription implements types.RecordData interface.
func (a *V2DataAdapter) GetDescription() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetDescription()
}

// GetAuthors implements types.RecordData interface.
func (a *V2DataAdapter) GetAuthors() []string {
	if a.record == nil {
		return nil
	}

	return a.record.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface.
func (a *V2DataAdapter) GetCreatedAt() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetCreatedAt()
}

// GetSkills implements types.RecordData interface.
func (a *V2DataAdapter) GetSkills() []types.Skill {
	if a.record == nil {
		return nil
	}

	skills := a.record.GetSkills()
	result := make([]types.Skill, len(skills))

	for i, skill := range skills {
		result[i] = NewV2SkillAdapter(skill)
	}

	return result
}

// GetLocators implements types.RecordData interface.
func (a *V2DataAdapter) GetLocators() []types.Locator {
	if a.record == nil {
		return nil
	}

	locators := a.record.GetLocators()
	result := make([]types.Locator, len(locators))

	for i, locator := range locators {
		result[i] = NewV2LocatorAdapter(locator)
	}

	return result
}

// GetExtensions implements types.RecordData interface.
func (a *V2DataAdapter) GetExtensions() []types.Extension {
	if a.record == nil {
		return nil
	}

	extensions := a.record.GetExtensions()
	result := make([]types.Extension, len(extensions))

	for i, extension := range extensions {
		result[i] = NewV2ExtensionAdapter(extension)
	}

	return result
}

// GetSignature implements types.RecordData interface.
func (a *V2DataAdapter) GetSignature() types.Signature {
	if a.record == nil || a.record.GetSignature() == nil {
		return nil
	}

	return NewV2SignatureAdapter(a.record.GetSignature())
}

// GetPreviousRecordCid implements types.RecordData interface.
func (a *V2DataAdapter) GetPreviousRecordCid() string {
	if a.record == nil {
		return ""
	}

	return a.record.GetPreviousAgentRecordCid()
}

// V2SignatureAdapter adapts objectsv2.Signature to types.Signature interface.
type V2SignatureAdapter struct {
	signature *objectsv2.Signature
}

// NewV2SignatureAdapter creates a new V2SignatureAdapter.
func NewV2SignatureAdapter(signature *objectsv2.Signature) *V2SignatureAdapter {
	return &V2SignatureAdapter{signature: signature}
}

// GetAnnotations implements types.Signature interface.
func (s *V2SignatureAdapter) GetAnnotations() map[string]string {
	if s.signature == nil {
		return nil
	}

	return s.signature.GetAnnotations()
}

// GetSignedAt implements types.Signature interface.
func (s *V2SignatureAdapter) GetSignedAt() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetSignedAt()
}

// GetAlgorithm implements types.Signature interface.
func (s *V2SignatureAdapter) GetAlgorithm() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetAlgorithm()
}

// GetSignature implements types.Signature interface.
func (s *V2SignatureAdapter) GetSignature() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetSignature()
}

// GetCertificate implements types.Signature interface.
func (s *V2SignatureAdapter) GetCertificate() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetCertificate()
}

// GetContentType implements types.Signature interface.
func (s *V2SignatureAdapter) GetContentType() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetContentType()
}

// GetContentBundle implements types.Signature interface.
func (s *V2SignatureAdapter) GetContentBundle() string {
	if s.signature == nil {
		return ""
	}

	return s.signature.GetContentBundle()
}

// V2ExtensionAdapter adapts objectsv2.Extension to types.Extension interface.
type V2ExtensionAdapter struct {
	extension *objectsv2.Extension
}

// NewV2ExtensionAdapter creates a new V2ExtensionAdapter.
func NewV2ExtensionAdapter(extension *objectsv2.Extension) *V2ExtensionAdapter {
	return &V2ExtensionAdapter{extension: extension}
}

// GetAnnotations implements types.Extension interface.
func (e *V2ExtensionAdapter) GetAnnotations() map[string]string {
	if e.extension == nil {
		return nil
	}

	return e.extension.GetAnnotations()
}

// GetName implements types.Extension interface.
func (e *V2ExtensionAdapter) GetName() string {
	if e.extension == nil {
		return ""
	}

	return e.extension.GetName()
}

// GetVersion implements types.Extension interface.
func (e *V2ExtensionAdapter) GetVersion() string {
	if e.extension == nil {
		return ""
	}

	return e.extension.GetVersion()
}

// GetData implements types.Extension interface.
func (e *V2ExtensionAdapter) GetData() map[string]any {
	if e.extension == nil || e.extension.GetData() == nil {
		return nil
	}

	return convertStructToMap(e.extension.GetData())
}

// V2SkillAdapter adapts objectsv2.Skill to types.Skill interface.
type V2SkillAdapter struct {
	skill *objectsv2.Skill
}

// NewV2SkillAdapter creates a new V2SkillAdapter.
func NewV2SkillAdapter(skill *objectsv2.Skill) *V2SkillAdapter {
	return &V2SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface.
func (s *V2SkillAdapter) GetAnnotations() map[string]string {
	if s.skill == nil {
		return nil
	}

	return s.skill.GetAnnotations()
}

// GetName implements types.Skill interface.
func (s *V2SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}

	return s.skill.GetName()
}

// GetID implements types.Skill interface.
func (s *V2SkillAdapter) GetID() uint64 {
	if s.skill == nil {
		return 0
	}

	return uint64(s.skill.GetId())
}

// V2LocatorAdapter adapts objectsv2.Locator to types.Locator interface.
type V2LocatorAdapter struct {
	locator *objectsv2.Locator
}

// NewV2LocatorAdapter creates a new V2LocatorAdapter.
func NewV2LocatorAdapter(locator *objectsv2.Locator) *V2LocatorAdapter {
	return &V2LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface.
func (l *V2LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}

	return l.locator.GetAnnotations()
}

// GetType implements types.Locator interface.
func (l *V2LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetType()
}

// GetURL implements types.Locator interface.
func (l *V2LocatorAdapter) GetURL() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetUrl()
}

// GetSize implements types.Locator interface.
func (l *V2LocatorAdapter) GetSize() uint64 {
	if l.locator == nil {
		return 0
	}

	return l.locator.GetSize()
}

// GetDigest implements types.Locator interface.
func (l *V2LocatorAdapter) GetDigest() string {
	if l.locator == nil {
		return ""
	}

	return l.locator.GetDigest()
}
