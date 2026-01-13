// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package naming

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/spf13/cobra"
)

var listOpts struct {
	limit int32
}

var listCmd = &cobra.Command{
	Use:   "list <domain>",
	Short: "List all verified agents for a domain",
	Long: `List all verified agents for a domain.

This command lists all records that have verified domain ownership for the
specified domain. This enables discovery of all agents published under a
domain (e.g., all agents from "cisco.com").

Usage examples:

1. List verified agents for a domain:
   dirctl naming list cisco.com

2. List with limit:
   dirctl naming list cisco.com --limit 10

3. List with JSON output:
   dirctl naming list cisco.com --output json

Note: This feature requires a domain index to be implemented on the server.
`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runListCommand(cmd, args[0])
	},
}

func init() {
	listCmd.Flags().Int32Var(&listOpts.limit, "limit", 100, "Maximum number of results to return")
}

func runListCommand(cmd *cobra.Command, domain string) error {
	// Get the client from the context
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return errors.New("failed to get client from context")
	}

	// Call ListVerifiedAgents
	resp, err := c.ListVerifiedAgents(cmd.Context(), domain, listOpts.limit)
	if err != nil {
		return fmt.Errorf("failed to list verified agents: %w", err)
	}

	agents := resp.GetAgents()
	if len(agents) == 0 {
		result := map[string]interface{}{
			"domain":  domain,
			"count":   0,
			"message": "No verified agents found for this domain",
		}

		return presenter.PrintMessage(cmd, "Verified Agents", "No verified agents found", result)
	}

	// Build result
	agentList := make([]map[string]interface{}, len(agents))
	for i, agent := range agents {
		agentList[i] = map[string]interface{}{
			"cid":         agent.GetCid(),
			"name":        agent.GetName(),
			"version":     agent.GetVersion(),
			"method":      agent.GetMethod(),
			"verified_at": agent.GetVerifiedAt().AsTime().Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	result := map[string]interface{}{
		"domain": domain,
		"count":  len(agents),
		"agents": agentList,
	}

	return presenter.PrintMessage(cmd, "Verified Agents", fmt.Sprintf("Found %d verified agents", len(agents)), result)
}
