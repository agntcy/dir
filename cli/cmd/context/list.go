// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured contexts",
	Long: `List configured dirctl client contexts.

This command reads the reusable client config file and prints each configured
context in sorted order. The persisted current_context is marked with '*'.

This command does not contact a Directory server and does not require a
resolved client configuration.

Examples:

1. List all configured contexts:
   dirctl context list

2. Use the marker to see which context is persisted as current_context:
   * prod
     staging`,
	Args: cobra.NoArgs,
	RunE: runList,
}

func runList(cmd *cobra.Command, _ []string) error {
	contexts, err := clientconfig.ListContexts("")
	if err != nil {
		return fmt.Errorf("failed to list contexts: %w", err)
	}

	if len(contexts) == 0 {
		cmd.Println("No contexts configured.")

		return nil
	}

	for _, contextSummary := range contexts {
		marker := " "
		if contextSummary.Current {
			marker = "*"
		}

		cmd.Printf("%s %s\n", marker, contextSummary.Name)
	}

	return nil
}
