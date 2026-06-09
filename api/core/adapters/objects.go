// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package adapters

import "google.golang.org/protobuf/types/known/structpb"

type module struct {
	Annotations map[string]string
	Name        string
	ID          uint64
	Data        *structpb.Struct
}

func (module *module) GetAnnotations() map[string]string { return module.Annotations }
func (module *module) GetName() string                   { return module.Name }
func (module *module) GetID() uint64                     { return module.ID }
func (module *module) GetData() *structpb.Struct         { return module.Data }

type skill struct {
	Annotations map[string]string
	Name        string
	ID          uint64
}

func (skill *skill) GetAnnotations() map[string]string { return skill.Annotations }
func (skill *skill) GetName() string                   { return skill.Name }
func (skill *skill) GetID() uint64                     { return skill.ID }

type domain struct {
	Annotations map[string]string
	Name        string
	ID          uint64
}

func (domain *domain) GetAnnotations() map[string]string { return domain.Annotations }
func (domain *domain) GetName() string                   { return domain.Name }
func (domain *domain) GetID() uint64                     { return domain.ID }

type locator struct {
	Annotations map[string]string
	Type        string
	URL         string
}

func (locator *locator) GetAnnotations() map[string]string { return locator.Annotations }
func (locator *locator) GetType() string                   { return locator.Type }
func (locator *locator) GetURL() string                    { return locator.URL }
