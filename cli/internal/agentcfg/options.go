// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

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
