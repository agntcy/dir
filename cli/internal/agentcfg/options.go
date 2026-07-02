// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

// Selection is one agent resolved for action, plus whether it was forced
// (explicitly selected or --force), which bypasses detection.
type Selection struct {
	Agent  Agent
	Forced bool
}

// ArtifactSet records which artifacts to act on.
type ArtifactSet struct {
	MCP   bool
	Skill bool
}

// ResolveArtifacts maps the --mcp/--skill flags to the artifact set. Neither or
// both flags means both artifacts.
func ResolveArtifacts(mcpFlag, skillFlag bool) ArtifactSet {
	if mcpFlag == skillFlag {
		return ArtifactSet{MCP: true, Skill: true}
	}

	return ArtifactSet{MCP: mcpFlag, Skill: skillFlag}
}

// ResolveSelection decides which agents to act on:
//   - explicit per-agent flags → exactly those, forced (detection bypassed);
//   - --force without explicit flags → all known agents, forced;
//   - default or --all → detected agents only.
func ResolveSelection(agents []Agent, env Env, chosen map[string]bool, all, force bool) []Selection {
	explicit := len(chosen) > 0

	var out []Selection

	for _, agent := range agents {
		switch {
		case explicit:
			if chosen[agent.ID] {
				out = append(out, Selection{Agent: agent, Forced: true})
			}
		case force:
			out = append(out, Selection{Agent: agent, Forced: true})
		default: // default and --all both act on detected agents
			_ = all

			if agent.Detect(env) {
				out = append(out, Selection{Agent: agent, Forced: false})
			}
		}
	}

	return out
}
