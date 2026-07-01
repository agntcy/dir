// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// nolint
package consumer

import (
	"context"
	"fmt"

	catalogv1 "github.com/agntcy/dir/api/catalog/v1"
	adsclient "github.com/agntcy/dir/client"
)

// Handle discovers, verifies, and dispatches entries matching a task query.
func Handle(ctx context.Context, client *adsclient.Client, taskQuery string) {
	entries, err := Discover(ctx)
	if err != nil {
		return
	}

	for _, e := range entries {
		if err := Verify(ctx, client, e); err != nil {
			continue // skip anything we can't verify
		}

		fmt.Printf("Discovered and verified entry: %s\n", e.Identifier)

		Dispatch(ctx, e, map[string]any{"task": taskQuery})
	}
}

// Dispatch routes a discovered entry to its native invocation path.
func Dispatch(ctx context.Context, e *catalogv1.CatalogEntry, task map[string]any) {
	switch e.GetMediaType() {
	case catalogv1.ProtocolMCPCardJsonMediaType:
		// Fetch the MCP server descriptor at e.URL, then speak JSON-RPC.
		fmt.Printf("Invoking MCP server: %s\n", e.Identifier)
	case catalogv1.ProtocolA2ACardJsonMediaType:
		// Load the A2A agent card at e.URL, then speak A2A.
		fmt.Printf("Invoking A2A agent: %s\n", e.Identifier)
	case catalogv1.ProtocolAgentSkillsMdMediaType:
		// Load the Agent Skill at e.URL
		fmt.Printf("Invoking Agent Skill: %s\n", e.Identifier)
	case catalogv1.ProtocolAgentSkillsBundleMediaType:
		// Load the bundled Agent Skill archive at e.URL
		fmt.Printf("Invoking Agent Skill bundle: %s\n", e.Identifier)
	default:
		fmt.Printf("Unsupported media type: %s\n", e.GetMediaType())
	}
}
