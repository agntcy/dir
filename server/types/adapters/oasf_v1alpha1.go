// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/core/converter"
)

// V1Alpha1Adapter adapts typesv1alpha1.Record to types.RecordData interface.
type V1Alpha1Adapter struct {
	record *typesv1alpha1.Record
}

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

// GetExtensions implements types.RecordData interface.
func (a *V1Alpha1Adapter) GetExtensions() []types.Extension {
	if a.record == nil {
		return nil
	}

	extensions := a.record.GetModules()
	result := make([]types.Extension, len(extensions))

	for i, extension := range extensions {
		result[i] = NewV1Alpha1ExtensionAdapter(extension)
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

// V1Alpha1ExtensionAdapter adapts typesv1alpha1.Extension to types.Extension interface.
type V1Alpha1ExtensionAdapter struct {
	extension *typesv1alpha1.Module
}

// NewV1Alpha1ExtensionAdapter creates a new V1Alpha1ExtensionAdapter.
func NewV1Alpha1ExtensionAdapter(extension *typesv1alpha1.Module) *V1Alpha1ExtensionAdapter {
	return &V1Alpha1ExtensionAdapter{extension: extension}
}

// GetAnnotations implements types.Extension interface.
func (e *V1Alpha1ExtensionAdapter) GetAnnotations() map[string]string {
	if e.extension == nil {
		return nil
	}

	return e.extension.GetAnnotations()
}

// GetName implements types.Extension interface.
func (e *V1Alpha1ExtensionAdapter) GetName() string {
	if e.extension == nil {
		return ""
	}

	return e.extension.GetName()
}

// GetVersion implements types.Extension interface.
func (e *V1Alpha1ExtensionAdapter) GetVersion() string {
	if e.extension == nil {
		return ""
	}

	// TODO: not implemented
	return ""
}

// GetData implements types.Extension interface.
func (e *V1Alpha1ExtensionAdapter) GetData() map[string]any {
	if e.extension == nil || e.extension.GetData() == nil {
		return nil
	}

	resp, err := converter.ProtoToStruct[map[string]any](e.extension.GetData())
	if err != nil {
		return nil
	}

	return *resp
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
