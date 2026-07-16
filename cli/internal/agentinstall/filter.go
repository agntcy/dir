// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

// SkillOnly returns a copy of the artifacts carrying only the skill (SKILL.md or
// bundle), with no MCP servers. Used to place the skill independently of the MCP
// server entry — e.g. `dirctl init` prompts for each separately.
func (a Artifacts) SkillOnly() Artifacts {
	return Artifacts{
		slug:        a.slug,
		skill:       a.skill,
		skillBundle: a.skillBundle,
	}
}

// MCPOnly returns a copy of the artifacts carrying only the MCP servers, with no
// skill. The counterpart to SkillOnly.
func (a Artifacts) MCPOnly() Artifacts {
	return Artifacts{
		slug:       a.slug,
		mcpServers: a.mcpServers,
	}
}
