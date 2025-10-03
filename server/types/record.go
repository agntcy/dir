// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type Record interface {
	GetCid() string
	GetRecordData() (RecordData, error)
}

type RecordMeta interface {
	GetCid() string
	GetAnnotations() map[string]string
	GetSchemaVersion() string
	GetCreatedAt() string
}

type RecordRef interface {
	GetCid() string
}

// Core abstraction interfaces.
//
//nolint:interfacebloat // RecordData is a cohesive interface for all record data operations
type RecordData interface {
	GetAnnotations() map[string]string
	GetSchemaVersion() string
	GetName() string
	GetVersion() string
	GetDescription() string
	GetAuthors() []string
	GetCreatedAt() string
	GetSkills() []Skill
	GetLocators() []Locator
	GetDomains() []Domain
	GetModules() []Module
	GetSignature() Signature
	GetPreviousRecordCid() string
}

type Signature interface {
	GetAnnotations() map[string]string
	GetSignedAt() string
	GetAlgorithm() string
	GetSignature() string
	GetCertificate() string
	GetContentType() string
	GetContentBundle() string
}

type Module interface {
	GetName() string
	GetData() map[string]any
}

//nolint:iface
type Skill interface {
	GetAnnotations() map[string]string
	GetName() string
	GetID() uint64
}

//nolint:iface
type Domain interface {
	GetAnnotations() map[string]string
	GetName() string
	GetID() uint64
}

type Locator interface {
	GetAnnotations() map[string]string
	GetType() string
	GetURL() string
	GetSize() uint64
	GetDigest() string
}

// LabelProvider provides routing labels for different record components.
// Implementations should handle version-specific label generation logic.
type LabelProvider interface {
	GetSkillLabels() []Label
	GetDomainLabels() []Label
	GetModuleLabels() []Label
	GetLocatorLabels() []Label
	GetAllLabels() []Label
}
