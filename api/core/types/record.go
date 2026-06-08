// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import "google.golang.org/protobuf/types/known/structpb"

// RecordReader is an interface that defines methods to read a record's data.
// It abstracts the underlying data structure of a record and allows for different
// implementations to provide access to the record's information.
type RecordReader interface {
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

// Module defines the necessary data for a module, which is a component of a record.
// It includes annotations, name, ID, and data.
type Module struct {
	Annotations map[string]string
	Name        string
	ID          uint64
	Data        *structpb.Struct
}

type Skill struct {
	Annotations map[string]string
	Name        string
	ID          uint64
}

type Domain struct {
	Annotations map[string]string
	Name        string
	ID          uint64
}

type Locator struct {
	Annotations map[string]string
	Type        string
	URLs        []string
}
