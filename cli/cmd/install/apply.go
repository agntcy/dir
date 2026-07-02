// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"maps"

	"github.com/agntcy/dir/cli/internal/agentcfg"
)

// runInstall applies the record's artifacts to the selected agents, one outcome
// per touched artifact. Errors on one agent never abort the rest.
func runInstall(env agentcfg.Env, arts artifacts, agents []agentcfg.Agent, set agentcfg.ArtifactSet, dryRun bool) []agentcfg.Outcome {
	var outcomes []agentcfg.Outcome

	seenSkill := map[string]bool{}

	for _, agent := range agents {
		if set.MCP && agent.MCP != nil {
			for _, srv := range arts.mcpServers {
				entry := styleEntry(srv.entry, agent.MCP.EntryStyle)
				o, _ := agentcfg.InstallMCP(agent.MCP, env, entry, srv.name, dryRun)
				o.Agent = agent.Name
				outcomes = append(outcomes, o)
			}
		}

		if set.Skill && agent.Skill != nil && arts.skill != "" {
			if dedupeSkill(seenSkill, agent.Skill, env, arts.slug) {
				continue
			}

			o, _ := agentcfg.InstallSkill(agent.Skill, env, arts.slug, arts.skill, dryRun)
			o.Agent = agent.Name
			outcomes = append(outcomes, o)
		}
	}

	return outcomes
}

// runUninstall removes the record's artifacts from the selected agents.
func runUninstall(env agentcfg.Env, arts artifacts, agents []agentcfg.Agent, set agentcfg.ArtifactSet, dryRun bool) []agentcfg.Outcome {
	var outcomes []agentcfg.Outcome

	seenSkill := map[string]bool{}

	for _, agent := range agents {
		if set.MCP && agent.MCP != nil {
			for _, srv := range arts.mcpServers {
				o, _ := agentcfg.RemoveMCP(agent.MCP, env, srv.name, dryRun)
				o.Agent = agent.Name
				outcomes = append(outcomes, o)
			}
		}

		if set.Skill && agent.Skill != nil && arts.skill != "" {
			if dedupeSkill(seenSkill, agent.Skill, env, arts.slug) {
				continue
			}

			o, _ := agentcfg.RemoveSkill(agent.Skill, env, arts.slug, dryRun)
			o.Agent = agent.Name
			outcomes = append(outcomes, o)
		}
	}

	return outcomes
}

// styleEntry clones base and applies agent-specific entry shaping (Zed adds a
// "source" field to its context_servers value).
func styleEntry(base map[string]any, style agentcfg.EntryStyle) map[string]any {
	entry := make(map[string]any, len(base)+1)
	maps.Copy(entry, base)

	if style == agentcfg.ZedContextServer {
		entry["source"] = "custom"
	}

	return entry
}

// dedupeSkill reports whether a skill target's resolved path was already acted on
// this run (e.g. Claude Code and Claude Desktop share one skills folder).
func dedupeSkill(seen map[string]bool, target *agentcfg.SkillTarget, env agentcfg.Env, slug string) bool {
	path, _, err := agentcfg.ResolveSkillTargetPath(target, env, slug)
	if err != nil || path == "" {
		return false
	}

	if seen[path] {
		return true
	}

	seen[path] = true

	return false
}
