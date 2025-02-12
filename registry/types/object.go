// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"fmt"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	registrytypes "github.com/agntcy/dir/api/registry/v1alpha1"
)

type ObjectConverter interface {
	Agent(*coretypes.Agent) (registrytypes.ObjectMeta, error)
	Locator(*coretypes.Locator) (registrytypes.ObjectMeta, error)
	Extension(*coretypes.Extension) (registrytypes.ObjectMeta, error)
}

type objectConverter struct{}

func NewObjectConverter() ObjectConverter {
	return &objectConverter{}
}

func convertDigest(d *coretypes.Digest) *coretypes.Digest {
	return &coretypes.Digest{
		Type:  coretypes.DigestType(d.Type),
		Value: d.Value,
	}
}

func (c *objectConverter) Agent(agent *coretypes.Agent) (registrytypes.ObjectMeta, error) {
	return registrytypes.ObjectMeta{
		Type:   registrytypes.ObjectType_OBJECT_TYPE_AGENT,
		Name:   fmt.Sprintf("%s-%s", agent.Name, agent.Version),
		Digest: convertDigest(agent.Digest),
	}, nil
}

func (c *objectConverter) Locator(locator *coretypes.Locator) (registrytypes.ObjectMeta, error) {
	return registrytypes.ObjectMeta{
		Type:   registrytypes.ObjectType_OBJECT_TYPE_LOCATOR,
		Name:   locator.Name,
		Digest: convertDigest(locator.Digest),
	}, nil
}

func (c *objectConverter) Extension(extension *coretypes.Extension) (registrytypes.ObjectMeta, error) {
	return registrytypes.ObjectMeta{
		Type:   registrytypes.ObjectType_OBJECT_TYPE_EXTENSION,
		Name:   extension.Name,
		Digest: convertDigest(extension.Digest),
	}, nil
}
