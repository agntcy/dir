// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

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

// ResolveSelection returns the agents to act on. Detection is always required —
// an agent is never selected unless it is detected on this machine.
//
//   - When chosen is empty ("all"), every detected agent is selected.
//   - When chosen is non-empty, only the chosen agents that are detected are
//     selected; chosen agents that are not detected are returned as the second
//     result so the caller can report them.
func ResolveSelection(agents []Agent, env Env, chosen map[string]bool) ([]Agent, []string) {
	explicit := len(chosen) > 0

	var selected []Agent

	var skipped []string

	for _, agent := range agents {
		if explicit && !chosen[agent.ID] {
			continue
		}

		switch {
		case agent.Detect(env):
			selected = append(selected, agent)
		case explicit:
			skipped = append(skipped, agent.ID)
		}
	}

	return selected, skipped
}
