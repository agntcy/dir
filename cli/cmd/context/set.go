// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

var setCmd = &cobra.Command{
	Use:   "set <name>",
	Short: "Set the persisted active context",
	Long: `Set the persisted active dirctl client context.

This command updates current_context in the reusable client config file. The
target context must already exist in the config file.

Use this when switching your terminal session from one Directory node to
another as the default target. For one-off commands, prefer --context instead
of changing the persisted active context.

Arguments:

<name> is required and must match a configured context name.

Examples:

1. Switch to the prod context:
   dirctl context set prod

2. Run one command against staging without changing the persisted context:
   dirctl --context staging info <record-cid>`,
	Args: cobra.ExactArgs(1),
	RunE: runSet,
}

func runSet(cmd *cobra.Command, args []string) error {
	resolved, err := clientconfig.SetCurrentContext("", args[0])
	if err != nil {
		return fmt.Errorf("failed to set current context: %w", err)
	}

	cmd.Printf("Switched to context %q.\n", resolved.Name)

	return nil
}
