package adapters

import (
	objectsv1 "github.com/agntcy/dir/api/objects/v1"
	"github.com/agntcy/dir/server/types"
)

// V1DataAdapter adapts objectsv1.Agent to types.RecordData interface
type V1DataAdapter struct {
	agent *objectsv1.Agent
}

// NewV1DataAdapter creates a new V1DataAdapter
func NewV1DataAdapter(agent *objectsv1.Agent) *V1DataAdapter {
	return &V1DataAdapter{agent: agent}
}

// GetAnnotations implements types.RecordData interface
func (a *V1DataAdapter) GetAnnotations() map[string]string {
	if a.agent == nil {
		return nil
	}
	return a.agent.GetAnnotations()
}

// GetSchemaVersion implements types.RecordData interface
func (a *V1DataAdapter) GetSchemaVersion() string {
	if a.agent == nil {
		return ""
	}
	return a.agent.GetSchemaVersion()
}

// GetName implements types.RecordData interface
func (a *V1DataAdapter) GetName() string {
	if a.agent == nil {
		return ""
	}
	return a.agent.GetName()
}

// GetVersion implements types.RecordData interface
func (a *V1DataAdapter) GetVersion() string {
	if a.agent == nil {
		return ""
	}
	return a.agent.GetVersion()
}

// GetDescription implements types.RecordData interface
func (a *V1DataAdapter) GetDescription() string {
	if a.agent == nil {
		return ""
	}
	return a.agent.GetDescription()
}

// GetAuthors implements types.RecordData interface
func (a *V1DataAdapter) GetAuthors() []string {
	if a.agent == nil {
		return nil
	}
	return a.agent.GetAuthors()
}

// GetCreatedAt implements types.RecordData interface
func (a *V1DataAdapter) GetCreatedAt() string {
	if a.agent == nil {
		return ""
	}
	return a.agent.GetCreatedAt()
}

// GetSkills implements types.RecordData interface
func (a *V1DataAdapter) GetSkills() []types.Skill {
	if a.agent == nil {
		return nil
	}
	skills := a.agent.GetSkills()
	result := make([]types.Skill, len(skills))
	for i, skill := range skills {
		result[i] = NewV1SkillAdapter(skill)
	}
	return result
}

// GetLocators implements types.RecordData interface
func (a *V1DataAdapter) GetLocators() []types.Locator {
	if a.agent == nil {
		return nil
	}
	locators := a.agent.GetLocators()
	result := make([]types.Locator, len(locators))
	for i, locator := range locators {
		result[i] = NewV1LocatorAdapter(locator)
	}
	return result
}

// GetExtensions implements types.RecordData interface
func (a *V1DataAdapter) GetExtensions() []types.Extension {
	if a.agent == nil {
		return nil
	}
	extensions := a.agent.GetExtensions()
	result := make([]types.Extension, len(extensions))
	for i, extension := range extensions {
		result[i] = NewV1ExtensionAdapter(extension)
	}
	return result
}

// GetSignature implements types.RecordData interface
func (a *V1DataAdapter) GetSignature() types.Signature {
	if a.agent == nil || a.agent.GetSignature() == nil {
		return nil
	}
	return NewV1SignatureAdapter(a.agent.GetSignature())
}

// GetPreviousRecordCid implements types.RecordData interface
func (a *V1DataAdapter) GetPreviousRecordCid() string {
	// V1 doesn't have previous record CID
	return ""
}

// V1SignatureAdapter adapts objectsv1.Signature to types.Signature interface
type V1SignatureAdapter struct {
	signature *objectsv1.Signature
}

// NewV1SignatureAdapter creates a new V1SignatureAdapter
func NewV1SignatureAdapter(signature *objectsv1.Signature) *V1SignatureAdapter {
	return &V1SignatureAdapter{signature: signature}
}

// GetAnnotations implements types.Signature interface
func (s *V1SignatureAdapter) GetAnnotations() map[string]string {
	// V1 signature doesn't have annotations
	return nil
}

// GetSignedAt implements types.Signature interface
func (s *V1SignatureAdapter) GetSignedAt() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetSignedAt()
}

// GetAlgorithm implements types.Signature interface
func (s *V1SignatureAdapter) GetAlgorithm() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetAlgorithm()
}

// GetSignature implements types.Signature interface
func (s *V1SignatureAdapter) GetSignature() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetSignature()
}

// GetCertificate implements types.Signature interface
func (s *V1SignatureAdapter) GetCertificate() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetCertificate()
}

// GetContentType implements types.Signature interface
func (s *V1SignatureAdapter) GetContentType() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetContentType()
}

// GetContentBundle implements types.Signature interface
func (s *V1SignatureAdapter) GetContentBundle() string {
	if s.signature == nil {
		return ""
	}
	return s.signature.GetContentBundle()
}

// V1ExtensionAdapter adapts objectsv1.Extension to types.Extension interface
type V1ExtensionAdapter struct {
	extension *objectsv1.Extension
}

// NewV1ExtensionAdapter creates a new V1ExtensionAdapter
func NewV1ExtensionAdapter(extension *objectsv1.Extension) *V1ExtensionAdapter {
	return &V1ExtensionAdapter{extension: extension}
}

// GetAnnotations implements types.Extension interface
func (e *V1ExtensionAdapter) GetAnnotations() map[string]string {
	if e.extension == nil {
		return nil
	}
	return e.extension.GetAnnotations()
}

// GetName implements types.Extension interface
func (e *V1ExtensionAdapter) GetName() string {
	if e.extension == nil {
		return ""
	}
	return e.extension.GetName()
}

// GetVersion implements types.Extension interface
func (e *V1ExtensionAdapter) GetVersion() string {
	if e.extension == nil {
		return ""
	}
	return e.extension.GetVersion()
}

// GetData implements types.Extension interface
func (e *V1ExtensionAdapter) GetData() map[string]any {
	if e.extension == nil || e.extension.GetData() == nil {
		return nil
	}
	return convertStructToMap(e.extension.GetData())
}

// V1SkillAdapter adapts objectsv1.Skill to types.Skill interface
type V1SkillAdapter struct {
	skill *objectsv1.Skill
}

// NewV1SkillAdapter creates a new V1SkillAdapter
func NewV1SkillAdapter(skill *objectsv1.Skill) *V1SkillAdapter {
	return &V1SkillAdapter{skill: skill}
}

// GetAnnotations implements types.Skill interface
func (s *V1SkillAdapter) GetAnnotations() map[string]string {
	if s.skill == nil {
		return nil
	}
	return s.skill.GetAnnotations()
}

// GetName implements types.Skill interface
func (s *V1SkillAdapter) GetName() string {
	if s.skill == nil {
		return ""
	}
	// TODO: maybe we should use the categoryName/className instead of just className
	return s.skill.GetClassName()
}

// GetId implements types.Skill interface
func (s *V1SkillAdapter) GetId() uint64 {
	if s.skill == nil {
		return 0
	}
	return s.skill.GetClassUid()
}

// V1LocatorAdapter adapts objectsv1.Locator to types.Locator interface
type V1LocatorAdapter struct {
	locator *objectsv1.Locator
}

// NewV1LocatorAdapter creates a new V1LocatorAdapter
func NewV1LocatorAdapter(locator *objectsv1.Locator) *V1LocatorAdapter {
	return &V1LocatorAdapter{locator: locator}
}

// GetAnnotations implements types.Locator interface
func (l *V1LocatorAdapter) GetAnnotations() map[string]string {
	if l.locator == nil {
		return nil
	}
	return l.locator.GetAnnotations()
}

// GetType implements types.Locator interface
func (l *V1LocatorAdapter) GetType() string {
	if l.locator == nil {
		return ""
	}
	return l.locator.GetType()
}

// GetUrl implements types.Locator interface
func (l *V1LocatorAdapter) GetUrl() string {
	if l.locator == nil {
		return ""
	}
	return l.locator.GetUrl()
}

// GetSize implements types.Locator interface
func (l *V1LocatorAdapter) GetSize() uint64 {
	if l.locator == nil {
		return 0
	}
	return l.locator.GetSize()
}

// GetDigest implements types.Locator interface
func (l *V1LocatorAdapter) GetDigest() string {
	if l.locator == nil {
		return ""
	}
	return l.locator.GetDigest()
}
