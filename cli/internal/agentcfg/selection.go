// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"strings"
)

// AllAgents is the --agents sentinel selecting every detected agent.
const AllAgents = "all"

// AgentIDs returns the known agent IDs from the registry, for flag help and
// validation.
func AgentIDs() []string {
	agents := Registry()
	ids := make([]string, 0, len(agents))

	for _, a := range agents {
		ids = append(ids, a.ID)
	}

	return ids
}

// ParseSelection validates raw --agents values and returns the chosen agent-ID
// set. An empty result means "all detected agents". It errors on an unknown ID
// or on combining AllAgents with specific IDs.
func ParseSelection(values []string) (map[string]bool, error) {
	valid := map[string]bool{}
	for _, id := range AgentIDs() {
		valid[id] = true
	}

	chosen := map[string]bool{}
	allMode := false

	for _, raw := range values {
		id := strings.TrimSpace(raw)

		switch {
		case id == AllAgents:
			allMode = true
		case valid[id]:
			chosen[id] = true
		default:
			return nil, fmt.Errorf("unknown agent %q; valid values: %s, %s",
				id, AllAgents, strings.Join(AgentIDs(), ", "))
		}
	}

	if allMode && len(chosen) > 0 {
		return nil, fmt.Errorf("--agents: %q cannot be combined with specific agent IDs", AllAgents)
	}

	return chosen, nil
}
