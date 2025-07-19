// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	coretypes "github.com/agntcy/dir/api/objects/v1"
	"github.com/agntcy/dir/server/types"
)

type AgentAdapter struct {
	*coretypes.Agent
	cid string
}

func NewAgentAdapter(agent *coretypes.Agent, cid string) *AgentAdapter {
	return &AgentAdapter{
		Agent: agent,
		cid:   cid,
	}
}

func (a *AgentAdapter) GetName() string {
	return a.Agent.GetName()
}

func (a *AgentAdapter) GetVersion() string {
	return a.Agent.GetVersion()
}

func (a *AgentAdapter) GetCID() string {
	return a.cid
}

func (a *AgentAdapter) GetSkillObjects() []types.SkillObject {
	skills := make([]types.SkillObject, 0, len(a.Agent.GetSkills()))
	for _, skill := range a.Agent.GetSkills() {
		skills = append(skills, skill)
	}

	return skills
}

func (a *AgentAdapter) GetLocatorObjects() []types.LocatorObject {
	locators := make([]types.LocatorObject, 0, len(a.Agent.GetLocators()))
	for _, locator := range a.Agent.GetLocators() {
		locators = append(locators, locator)
	}

	return locators
}

func (a *AgentAdapter) GetExtensionObjects() []types.ExtensionObject {
	extensions := make([]types.ExtensionObject, 0, len(a.Agent.GetExtensions()))
	for _, extension := range a.Agent.GetExtensions() {
		extensions = append(extensions, extension)
	}

	return extensions
}
