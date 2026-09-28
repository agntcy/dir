// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
)

const (
	orphanedServerReason = "the new version no longer defines this server"
	orphanedSkillReason  = "the new version ships no skill"
)

// RemoveOrphans removes the artifacts a manifest row records that the
// replacement record no longer provides.
//
// This is the step of an upgrade that installing the new version cannot do for
// itself, and the reason the row is consulted rather than the new record. Two
// cases produce an orphan:
//
//   - A renamed MCP server. Installing v2 adds its own key and knows nothing
//     of v1's, which sits in a config file dirctl shares with the user and
//     other tools and so cannot simply replace. Only the row can say which key
//     was ours.
//   - A dropped skill. v2 carries no agent_skills module, so installing it
//     writes no skill at all and v1's folder would survive with nothing to
//     replace it.
//
// Everything else the install handles by itself: a skill folder is dirctl's
// alone, so writing one replaces its entire contents.
//
// Each row resolves its own paths, so a `--project` row is pruned in the
// repository it names rather than wherever the command was run.
func RemoveOrphans(env agentcfg.Env, entries []pkgstate.Entry, arts Artifacts, dryRun bool) []agentcfg.Outcome {
	kept := make(map[string]bool)
	for _, name := range arts.MCPServerNames() {
		kept[name] = true
	}

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

		scope, rowEnv := placement(entry, env)

		outcomes = append(outcomes, removeOrphanedMCP(agent, rowEnv, entry, kept, scope, dryRun)...)

		if o, ok := removeOrphanedSkill(agent, rowEnv, entry, arts, scope, dryRun, seenSkill); ok {
			outcomes = append(outcomes, o)
		}
	}

	return outcomes
}

func removeOrphanedMCP(
	agent agentcfg.Agent,
	env agentcfg.Env,
	entry pkgstate.Entry,
	kept map[string]bool,
	scope agentcfg.Scope,
	dryRun bool,
) []agentcfg.Outcome {
	if agent.MCP == nil {
		return nil
	}

	var outcomes []agentcfg.Outcome

	for _, server := range entry.MCPServers {
		if kept[server] {
			// The new version defines this key too, so installing it is the
			// update. Removing it first would only report a change twice.
			continue
		}

		o, _ := agentcfg.RemoveMCP(agent.MCP, env, server, scope, dryRun)
		o.Agent = agent.Name
		o.Server = server

		if o.Reason == "" {
			o.Reason = orphanedServerReason
		}

		outcomes = append(outcomes, o)
	}

	return outcomes
}

func removeOrphanedSkill(
	agent agentcfg.Agent,
	env agentcfg.Env,
	entry pkgstate.Entry,
	arts Artifacts,
	scope agentcfg.Scope,
	dryRun bool,
	seenSkill map[string]bool,
) (agentcfg.Outcome, bool) {
	if entry.SkillPath == "" || agent.Skill == nil || arts.hasSkill() {
		return agentcfg.Outcome{}, false
	}

	slug := sanitizeSlug(entry.Name)

	// Claude Code and Claude Desktop share one skills folder, as do Zed and
	// Codex CLI. Removing it once is enough, but both rows still need an
	// outcome so both stop naming it.
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

	if o.Reason == "" {
		o.Reason = orphanedSkillReason
	}

	return o, true
}

// Placement maps a manifest row onto the placement engine's scope and the
// environment that resolves its paths, so the command layer can install into
// the location a row already names. See placement.
func Placement(entry pkgstate.Entry, env agentcfg.Env) (agentcfg.Scope, agentcfg.Env) {
	return placement(entry, env)
}

// Pruned reports what an orphan prune took away from one agent: whether its
// skill is gone, and which server keys are. It is what keeps a reconciled
// manifest row from carrying forward an artifact that was just removed.
//
// ActionUnchanged counts as gone: for a removal it means the artifact was
// looked at and found already absent, which is as good a confirmation as
// removing it. A skip or a failure counts for nothing, because in both cases
// something of ours may well still be on disk.
func Pruned(agent agentcfg.Agent, outcomes []agentcfg.Outcome) (bool, []string) {
	skill := false

	var servers []string

	for _, o := range outcomes {
		if o.Agent != agent.Name || !cleared(o.Action) {
			continue
		}

		switch o.Artifact {
		case agentcfg.ArtifactSkill:
			skill = true
		case agentcfg.ArtifactMCP:
			if o.Server != "" {
				servers = append(servers, o.Server)
			}
		}
	}

	return skill, servers
}

// cleared reports whether an action confirms our artifact is no longer there.
func cleared(action agentcfg.Action) bool {
	switch action {
	case agentcfg.ActionRemoved, agentcfg.ActionUnchanged:
		return true
	case agentcfg.ActionAdded, agentcfg.ActionUpdated, agentcfg.ActionSkipped, agentcfg.ActionFailed:
		return false
	default:
		return false
	}
}
