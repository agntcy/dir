// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

type Record interface {
	GetCid() string
	GetRecordData() RecordData
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

// Core abstraction interfaces
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
	GetExtensions() []Extension
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

type Extension interface {
	GetAnnotations() map[string]string
	GetName() string
	GetVersion() string
	GetData() map[string]any
}

type Skill interface {
	GetAnnotations() map[string]string
	GetName() string
	GetId() uint64
}

type Locator interface {
	GetAnnotations() map[string]string
	GetType() string
	GetUrl() string
	GetSize() uint64
	GetDigest() string
}
