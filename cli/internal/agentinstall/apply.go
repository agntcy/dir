// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"maps"

	"github.com/agntcy/dir/cli/internal/agentcfg"
)

const skillBundleFolderOnlyReason = "skill bundle requires a multi-file skills directory; this agent only supports single instruction files"

// Install applies the record's artifacts to the selected agents at the given
// scope, one outcome per touched artifact. Errors on one agent never abort the rest.
func Install(env agentcfg.Env, arts Artifacts, agents []agentcfg.Agent, scope agentcfg.Scope, dryRun bool) []agentcfg.Outcome {
	var outcomes []agentcfg.Outcome

	seenSkill := map[string]bool{}

	for _, agent := range agents {
		if agent.MCP != nil {
			for _, srv := range arts.mcpServers {
				entry := styleEntry(srv.entry, agent.MCP.EntryStyle)
				o, _ := agentcfg.InstallMCP(agent.MCP, env, entry, srv.name, scope, dryRun)
				o.Agent = agent.Name
				outcomes = append(outcomes, o)
			}
		}

		if agent.Skill != nil && arts.hasSkill() {
			if arts.hasSkillBundle() && agent.Skill.Strategy != agentcfg.SkillFolder {
				outcomes = append(outcomes, agentcfg.Outcome{
					Agent:    agent.Name,
					Artifact: "skill",
					Action:   agentcfg.ActionSkipped,
					Reason:   skillBundleFolderOnlyReason,
				})

				continue
			}

			if dedupeSkill(seenSkill, agent.Skill, env, arts.slug, scope) {
				continue
			}

			var o agentcfg.Outcome
			if arts.hasSkillBundle() {
				o, _ = agentcfg.InstallSkillBundle(agent.Skill, env, arts.slug, arts.skillBundle, scope, dryRun)
			} else {
				o, _ = agentcfg.InstallSkill(agent.Skill, env, arts.slug, arts.skill, scope, dryRun)
			}

			o.Agent = agent.Name
			outcomes = append(outcomes, o)
		}
	}

	return outcomes
}

// Uninstall removes the record's artifacts from the selected agents at the given scope.
func Uninstall(env agentcfg.Env, arts Artifacts, agents []agentcfg.Agent, scope agentcfg.Scope, dryRun bool) []agentcfg.Outcome {
	var outcomes []agentcfg.Outcome

	seenSkill := map[string]bool{}

	for _, agent := range agents {
		if agent.MCP != nil {
			for _, srv := range arts.mcpServers {
				o, _ := agentcfg.RemoveMCP(agent.MCP, env, srv.name, scope, dryRun)
				o.Agent = agent.Name
				outcomes = append(outcomes, o)
			}
		}

		if agent.Skill != nil && arts.hasSkill() {
			if arts.hasSkillBundle() && agent.Skill.Strategy != agentcfg.SkillFolder {
				continue
			}

			if dedupeSkill(seenSkill, agent.Skill, env, arts.slug, scope) {
				continue
			}

			o, _ := agentcfg.RemoveSkill(agent.Skill, env, arts.slug, scope, dryRun)
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
func dedupeSkill(seen map[string]bool, target *agentcfg.SkillTarget, env agentcfg.Env, slug string, scope agentcfg.Scope) bool {
	path, err := agentcfg.ResolveSkillTargetPath(target, env, slug, scope)
	if err != nil || path == "" {
		return false
	}

	if seen[path] {
		return true
	}

	seen[path] = true

	return false
}
