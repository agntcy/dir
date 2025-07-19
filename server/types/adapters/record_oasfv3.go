package adapters

import (
	objectsv3 "github.com/agntcy/dir/api/objects/v3"
	"github.com/agntcy/dir/server/types"
)

// V3DataAdapter adapts objectsv3.Record to types.RecordData interface
type V3DataAdapter struct {
	record *objectsv3.Record
}

// NewV3DataAdapter creates a new V3DataAdapter
func NewV3DataAdapter(record *objectsv3.Record) *V3DataAdapter {
	return &V3DataAdapter{record: record}
}

// GetAnnotations implements types.RecordData interface
func (a *V3DataAdapter) GetAnnotations() map[string]string {
	if a.record == nil {
		return nil
	}
	return a.record.GetAnnotations()
}

// GetSchemaVersion implements types.RecordData interface
func (a *V3DataAdapter) GetSchemaVersion() string {
	if a.record == nil {
		return ""
	}
	return a.record.GetSchemaVersion()
}

// GetName implements types.RecordData interface
func (a *V3DataAdapter) GetName() string {
	if a.record == nil {
		return ""
	}
	return a.record.GetName()
}

// GetVersion implements types.RecordData interface
func (a *V3DataAdapter) GetVersion() string {
	if a.record == nil {
		return ""
	}
	return a.record.GetVersion()
}

// GetDescription implements types.RecordData interface
func (a *V3DataAdapter) GetDescription() string {
	if a.record == nil {
		return ""
	}
	return a.record.GetDescription()
}

// GetAuthors implements types.RecordData interface
func (a *V3DataAdapter) GetAuthors() []string {
	if a.record == nil {
		return nil
	}
	return a.record.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface
func (a *V3DataAdapter) GetCreatedAt() string {
	if a.record == nil {
		return ""
	}
	return a.record.GetCreatedAt()
}

// GetSkills implements types.RecordData interface
func (a *V3DataAdapter) GetSkills() []types.Skill {
	if a.record == nil {
		return nil
	}
	skills := a.record.GetSkills()
	result := make([]types.Skill, len(skills))
	for i, skill := range skills {
		result[i] = NewV3SkillAdapter(skill)
	}
	return result
}

// GetLocators implements types.RecordData interface
func (a *V3DataAdapter) GetLocators() []types.Locator {
	if a.record == nil {
		return nil
	}
	locators := a.record.GetLocators()
	result := make([]types.Locator, len(locators))
	for i, locator := range locators {
		result[i] = NewV3LocatorAdapter(locator)
	}
	return result
}

// GetExtensions implements types.RecordData interface
func (a *V3DataAdapter) GetExtensions() []types.Extension {
	if a.record == nil {
		return nil
	}
	extensions := a.record.GetExtensions()
	result := make([]types.Extension, len(extensions))
	for i, extension := range extensions {
		result[i] = NewV3ExtensionAdapter(extension)
	}
	return result
}

// GetSignature implements types.RecordData interface
func (a *V3DataAdapter) GetSignature() types.Signature {
	if a.record == nil || a.record.GetSignature() == nil {
		return nil
	}
	return NewV3SignatureAdapter(a.record.GetSignature())
}

// GetPreviousRecordCid implements types.RecordData interface
func (a *V3DataAdapter) GetPreviousRecordCid() string {
	if a.record == nil {
		return ""
	}
	return a.record.GetPreviousRecordCid()
}

// V3SignatureAdapter adapts objectsv3.Signature to types.Signature interface
type V3SignatureAdapter struct {
	signature *objectsv3.Signature
}

// NewV3SignatureAdapter creates a new V3SignatureAdapter
func NewV3SignatureAdapter(signature *objectsv3.Signature) *V3SignatureAdapter {
	return &V3SignatureAdapter{signature: signature}
}

// GetAnnotations implements types.Signature interface
func (s *V3SignatureAdapter) GetAnnotations() map[string]string {
	if s.signature == nil {
		return nil
	}
	return s.signature.GetAnnotations()
}

// GetSignedAt implements types.Signature interface
func (s *V3SignatureAdapter) GetSignedAt() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetSignedAt()
}

// GetAlgorithm implements types.Signature interface
func (s *V3SignatureAdapter) GetAlgorithm() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetAlgorithm()
}

// GetSignature implements types.Signature interface
func (s *V3SignatureAdapter) GetSignature() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetSignature()
}

// GetCertificate implements types.Signature interface
func (s *V3SignatureAdapter) GetCertificate() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetCertificate()
}

// GetContentType implements types.Signature interface
func (s *V3SignatureAdapter) GetContentType() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetContentType()
}

// GetContentBundle implements types.Signature interface
func (s *V3SignatureAdapter) GetContentBundle() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetContentBundle()
}

// V3ExtensionAdapter adapts objectsv3.Extension to types.Extension interface
type V3ExtensionAdapter struct {
	extension *objectsv3.Extension
}

// NewV3ExtensionAdapter creates a new V3ExtensionAdapter
func NewV3ExtensionAdapter(extension *objectsv3.Extension) *V3ExtensionAdapter {
	return &V3ExtensionAdapter{extension: extension}
}

// GetAnnotations implements types.Extension interface
func (e *V3ExtensionAdapter) GetAnnotations() map[string]string {
	if e.extension == nil {
		return nil
	}
	return e.extension.GetAnnotations()
}

// GetName implements types.Extension interface
func (e *V3ExtensionAdapter) GetName() string {
	if e.extension == nil {
		return ""
	}
	return e.extension.GetName()
}

// GetVersion implements types.Extension interface
func (e *V3ExtensionAdapter) GetVersion() string {
	if e.extension == nil {
		return ""
	}
	return e.extension.GetVersion()
}

// GetData implements types.Extension interface
func (e *V3ExtensionAdapter) GetData() map[string]any {
	if e.extension == nil || e.extension.GetData() == nil {
		return nil
	}
	return convertStructToMap(e.extension.GetData())
}

// V3SkillAdapter adapts objectsv3.Skill to types.Skill interface
type V3SkillAdapter struct {
	skill *objectsv3.Skill
}

// NewV3SkillAdapter creates a new V3SkillAdapter
func NewV3SkillAdapter(skill *objectsv3.Skill) *V3SkillAdapter {
	return &V3SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface
func (s *V3SkillAdapter) GetAnnotations() map[string]string {
	if s.skill == nil {
		return nil
	}
	return s.skill.GetAnnotations()
}

// GetName implements types.Skill interface
func (s *V3SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}
	return s.skill.GetName()
}

// GetId implements types.Skill interface
func (s *V3SkillAdapter) GetId() uint64 {
	if s.skill == nil {
		return 0
	}
	return uint64(s.skill.GetId())
}

// V3LocatorAdapter adapts objectsv3.Locator to types.Locator interface
type V3LocatorAdapter struct {
	locator *objectsv3.Locator
}

// NewV3LocatorAdapter creates a new V3LocatorAdapter
func NewV3LocatorAdapter(locator *objectsv3.Locator) *V3LocatorAdapter {
	return &V3LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface
func (l *V3LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}
	return l.locator.GetAnnotations()
}

// GetType implements types.Locator interface
func (l *V3LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}
	return l.locator.GetType()
}

// GetUrl implements types.Locator interface
func (l *V3LocatorAdapter) GetUrl() string {
	if l.locator == nil {
		return ""
	}
	return l.locator.GetUrl()
}

// GetSize implements types.Locator interface
func (l *V3LocatorAdapter) GetSize() uint64 {
	if l.locator == nil {
		return 0
	}
	return l.locator.GetSize()
}

// GetDigest implements types.Locator interface
func (l *V3LocatorAdapter) GetDigest() string {
	if l.locator == nil {
		return ""
	}
	return l.locator.GetDigest()
}
