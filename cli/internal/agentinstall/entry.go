// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
)

// Slug is the sanitized record name that names the skill folder or file and the
// managed-block marker.
func (a Artifacts) Slug() string { return a.slug }

// HasSkill reports whether the record yields a skill artifact at all.
func (a Artifacts) HasSkill() bool { return a.hasSkill() }

// MCPServerNames lists the config keys the MCP server entries are stored under,
// in the sorted order DeriveArtifacts fixed so output is stable across runs.
func (a Artifacts) MCPServerNames() []string {
	names := make([]string, 0, len(a.mcpServers))
	for _, srv := range a.mcpServers {
		names = append(names, srv.name)
	}

	return names
}

// SkillFiles lists a skill bundle's files, relative to the installed skill
// folder. It is empty for a single-file skill, where the recorded skill path is
// itself the one file.
func (a Artifacts) SkillFiles() []string { return a.skillFiles }

// BuildEntry builds the install-manifest row for one agent out of the outcomes
// an apply produced, recording the artifacts that actually landed rather than
// the ones the record's modules imply. That distinction is the point: a later
// uninstall or upgrade removes exactly these paths and server keys, without
// re-fetching a record that may since have been garbage-collected upstream.
//
// It reports false when nothing landed for this agent, because every outcome
// was skipped or failed, so a listing never claims an install that did not
// happen. The caller fills in the record identity — name, version, CID, scope,
// origin, pin — and the timestamp, none of which Artifacts carries.
//
// Outcomes for other agents are ignored, so the whole apply result can be
// passed in for each agent in turn.
func BuildEntry(arts Artifacts, agent agentcfg.Agent, outcomes []agentcfg.Outcome) (pkgstate.Entry, bool) {
	entry := pkgstate.Entry{Agent: agent.ID}
	landed := false

	for _, o := range outcomes {
		if o.Agent != agent.Name || !wrote(o.Action) {
			continue
		}

		switch o.Artifact {
		case agentcfg.ArtifactSkill:
			entry.SkillPath = o.Path
			entry.SkillFiles = arts.SkillFiles()
			landed = true
		case agentcfg.ArtifactMCP:
			if o.Server != "" {
				entry.MCPServers = append(entry.MCPServers, o.Server)
			}

			landed = true
		}
	}

	if !landed {
		return pkgstate.Entry{}, false
	}

	return entry, true
}

// Cleared reports whether an uninstall left this agent with nothing of ours:
// it acted on at least one artifact and none of them failed. An agent whose
// removal failed keeps its manifest row, because its artifacts are still on
// disk and something still has to remove them.
func Cleared(agent agentcfg.Agent, outcomes []agentcfg.Outcome) bool {
	acted := false

	for _, o := range outcomes {
		if o.Agent != agent.Name {
			continue
		}

		if o.Action == agentcfg.ActionFailed {
			return false
		}

		acted = true
	}

	return acted
}

// wrote reports whether an action left our artifact in place: added and updated
// wrote it, unchanged found it already correct. Skipped and failed left nothing
// behind, and removed belongs to uninstall.
func wrote(action agentcfg.Action) bool {
	switch action {
	case agentcfg.ActionAdded, agentcfg.ActionUpdated, agentcfg.ActionUnchanged:
		return true
	case agentcfg.ActionRemoved, agentcfg.ActionSkipped, agentcfg.ActionFailed:
		return false
	default:
		return false
	}
}
