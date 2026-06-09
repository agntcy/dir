// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// Record is an interface that defines methods to access a record's data.
// It abstracts the underlying data structure of a record and allows for different
// implementations and versions to provide unified access to relevant information.
//
//nolint:interfacebloat
type Record interface {
	GetCid() string
	GetAnnotations() map[string]string
	GetSchemaVersion() string
	GetName() string
	GetVersion() string
	GetDescription() string
	GetAuthors() []string
	GetCreatedAt() string
	GetPreviousRecordCid() string
	GetSkills() []Skill
	GetLocators() []Locator
	GetDomains() []Domain
	GetModules() []Module
}

// Module defines the necessary data for a module.
type Module interface {
	GetAnnotations() map[string]string
	GetName() string
	GetID() uint64
	GetData() *structpb.Struct
}

// Skill defines the necessary data for a skill.
//
//nolint:iface
type Skill interface {
	GetAnnotations() map[string]string
	GetName() string
	GetID() uint64
}

// Domain defines the necessary data for a domain.
//
//nolint:iface
type Domain interface {
	GetAnnotations() map[string]string
	GetName() string
	GetID() uint64
}

// Locator defines the necessary data for a locator.
type Locator interface {
	GetAnnotations() map[string]string
	GetType() string
	GetURL() string
}
