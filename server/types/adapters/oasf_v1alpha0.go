// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"fmt"

	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha0"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/oasf-sdk/core/converter"
)

// V1Alpha0Adapter adapts typesv1alpha0.Record to types.RecordData interface.
type V1Alpha0Adapter struct {
	record *typesv1alpha0.Record
}

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

// GetExtensions implements types.RecordData interface.
func (a *V1Alpha0Adapter) GetExtensions() []types.Extension {
	if a.record == nil {
		return nil
	}

	extensions := a.record.GetExtensions()
	result := make([]types.Extension, len(extensions))

	for i, extension := range extensions {
		result[i] = NewV1Alpha0ExtensionAdapter(extension)
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

// V1Alpha0ExtensionAdapter adapts typesv1alpha0.Extension to types.Extension interface.
type V1Alpha0ExtensionAdapter struct {
	extension *typesv1alpha0.Extension
}

// NewV1Alpha0ExtensionAdapter creates a new V1Alpha0ExtensionAdapter.
func NewV1Alpha0ExtensionAdapter(extension *typesv1alpha0.Extension) *V1Alpha0ExtensionAdapter {
	return &V1Alpha0ExtensionAdapter{extension: extension}
}

// GetAnnotations implements types.Extension interface.
func (e *V1Alpha0ExtensionAdapter) GetAnnotations() map[string]string {
	if e.extension == nil {
		return nil
	}

	return e.extension.GetAnnotations()
}

// GetName implements types.Extension interface.
func (e *V1Alpha0ExtensionAdapter) GetName() string {
	if e.extension == nil {
		return ""
	}

	return e.extension.GetName()
}

// GetVersion implements types.Extension interface.
func (e *V1Alpha0ExtensionAdapter) GetVersion() string {
	if e.extension == nil {
		return ""
	}

	return e.extension.GetVersion()
}

// GetData implements types.Extension interface.
func (e *V1Alpha0ExtensionAdapter) GetData() map[string]any {
	if e.extension == nil || e.extension.GetData() == nil {
		return nil
	}

	resp, err := converter.ProtoToStruct[map[string]any](e.extension.GetData())
	if err != nil {
		return nil
	}

	return *resp
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
