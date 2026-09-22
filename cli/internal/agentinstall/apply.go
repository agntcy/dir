// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"maps"
	"path/filepath"

	"github.com/agntcy/dir/cli/internal/agentcfg"
)

const (
	skillBundleFolderOnlyReason = "skill bundle requires a multi-file skills directory; this agent only supports single instruction files"
	sharedSkillReason           = "shared skill location, written once for another agent"
)

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
					Artifact: agentcfg.ArtifactSkill,
					Action:   agentcfg.ActionSkipped,
					Reason:   skillBundleFolderOnlyReason,
				})

				continue
			}

			if path, shared := claimSkillPath(seenSkill, agent.Skill, env, arts.slug, scope); shared {
				outcomes = append(outcomes, sharedSkillOutcome(agent, path, arts))

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

			if path, shared := claimSkillPath(seenSkill, agent.Skill, env, arts.slug, scope); shared {
				outcomes = append(outcomes, sharedSkillOutcome(agent, path, arts))

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

// claimSkillPath marks a skill target's resolved path as acted on this run and
// reports the path, plus whether another agent had already claimed it.
//
// Claude Code and Claude Desktop share one skills folder, and so do Zed and
// Codex CLI, so the artifact is written once. The agents that share it are
// served by it all the same, which is why the caller reports an outcome for
// them rather than passing over them in silence: without one, the install
// manifest would hold a row for the first agent only, and uninstalling for
// just one of the pair could never clear the other's row.
func claimSkillPath(
	seen map[string]bool,
	target *agentcfg.SkillTarget,
	env agentcfg.Env,
	slug string,
	scope agentcfg.Scope,
) (string, bool) {
	path, err := agentcfg.ResolveSkillTargetPath(target, env, slug, scope)
	if err != nil || path == "" {
		return "", false
	}

	if seen[path] {
		return path, true
	}

	seen[path] = true

	return path, false
}

// sharedSkillOutcome reports an agent served by a skill that another agent in
// this run already wrote or removed. It carries the same path that agent's own
// outcome carried, so both manifest rows name the same artifact, and reports
// ActionUnchanged because nothing was written for this agent, which is also
// how an already-correct or already-absent artifact reports elsewhere.
func sharedSkillOutcome(agent agentcfg.Agent, path string, arts Artifacts) agentcfg.Outcome {
	// A bundle's outcome names the folder it was extracted into, not SKILL.md.
	if arts.hasSkillBundle() {
		path = filepath.Dir(path)
	}

	return agentcfg.Outcome{
		Agent:    agent.Name,
		Artifact: agentcfg.ArtifactSkill,
		Path:     path,
		Action:   agentcfg.ActionUnchanged,
		Reason:   sharedSkillReason,
	}
}
