// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
)

// unknownAgentReason explains a row naming an agent this binary has no
// descriptor for, so its config location cannot be resolved.
const unknownAgentReason = "this dirctl does not know this agent, so its config location cannot be resolved"

// UninstallRecorded removes the artifacts the given manifest rows name.
//
// This is the counterpart to Uninstall, and the difference is where the work
// list comes from. Uninstall derives artifacts from the record, so it needs the
// Directory to hand that record back, and it can only act on agents detected
// right now. UninstallRecorded reads the manifest, which means:
//
//   - No Directory call. A package can be removed with the server down, or
//     after its record has been deleted upstream.
//   - Only the agents that actually hold the package are touched, so the plan
//     names those and no others.
//   - An agent no longer detected on this machine still gets cleaned up, since
//     the files it was given are still on disk.
//
// The skill path is re-derived from the record name rather than read from the
// row, because removal has to know the agent's strategy — a folder, a single
// file, or a block inside a shared file — and the strategy comes from the
// registry along with the location. The row's recorded paths are what `list`
// reports and what a later file-precise removal will use.
func UninstallRecorded(env agentcfg.Env, entries []pkgstate.Entry, dryRun bool) []agentcfg.Outcome {
	var outcomes []agentcfg.Outcome

	seenSkill := map[string]bool{}

	for _, entry := range entries {
		agent, known := agentcfg.ByID(entry.Agent)
		if !known {
			outcomes = append(outcomes, agentcfg.Outcome{
				Agent:    entry.Agent,
				Artifact: agentcfg.ArtifactSkill,
				Action:   agentcfg.ActionSkipped,
				Reason:   unknownAgentReason,
			})

			continue
		}

		// A project row's paths resolve from the repository it names, not from
		// wherever this command was run.
		scope, rowEnv := placement(entry, env)
		slug := sanitizeSlug(entry.Name)

		outcomes = append(outcomes, removeRecordedMCP(agent, rowEnv, entry, scope, dryRun)...)

		if o, ok := removeRecordedSkill(agent, rowEnv, entry, slug, scope, dryRun, seenSkill); ok {
			outcomes = append(outcomes, o)
		}
	}

	return outcomes
}

func removeRecordedMCP(
	agent agentcfg.Agent,
	env agentcfg.Env,
	entry pkgstate.Entry,
	scope agentcfg.Scope,
	dryRun bool,
) []agentcfg.Outcome {
	if agent.MCP == nil {
		return nil
	}

	outcomes := make([]agentcfg.Outcome, 0, len(entry.MCPServers))

	for _, server := range entry.MCPServers {
		o, _ := agentcfg.RemoveMCP(agent.MCP, env, server, scope, dryRun)
		o.Agent = agent.Name
		outcomes = append(outcomes, o)
	}

	return outcomes
}

func removeRecordedSkill(
	agent agentcfg.Agent,
	env agentcfg.Env,
	entry pkgstate.Entry,
	slug string,
	scope agentcfg.Scope,
	dryRun bool,
	seenSkill map[string]bool,
) (agentcfg.Outcome, bool) {
	if entry.SkillPath == "" || agent.Skill == nil {
		return agentcfg.Outcome{}, false
	}

	// Claude Code and Claude Desktop share one skills folder, as do Zed and
	// Codex CLI. Removing it once is enough, but both rows still need an outcome
	// so both are dropped from the manifest.
	if path, shared := claimSkillPath(seenSkill, agent.Skill, env, slug, scope); shared {
		return agentcfg.Outcome{
			Agent:    agent.Name,
			Artifact: agentcfg.ArtifactSkill,
			Path:     path,
			Action:   agentcfg.ActionUnchanged,
			Reason:   sharedSkillReason,
		}, true
	}

	o, _ := agentcfg.RemoveSkill(agent.Skill, env, slug, scope, dryRun)
	o.Agent = agent.Name

	return o, true
}
