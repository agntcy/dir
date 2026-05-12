// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"github.com/spf13/cobra"
)

// Command is the parent command for reusable client contexts.
var Command = &cobra.Command{
	Use:   "context",
	Short: "Manage client contexts",
	Long: `Manage reusable dirctl client contexts.

Contexts live in the dirctl client config file and describe Directory nodes,
authentication settings, and TLS/SPIFFE settings.

This command group helps users operate against multiple Directory nodes from
one terminal by inspecting configured contexts, switching the persisted active
context, showing the effective client configuration, and validating context
definitions before use.

Available operations:

- list: Show all configured contexts and mark the persisted active context
- current: Print the persisted current_context from config
- set: Persist a configured context as the active context
- show: Display the effective context with sensitive values redacted
- validate: Validate one context or all configured contexts

Examples:

1. List all contexts:
   dirctl context list

2. Show the active context name:
   dirctl context current

3. Switch the persisted active context:
   dirctl context set prod

4. Show an effective context:
   dirctl context show dev

5. Validate all contexts:
   dirctl context validate`,
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		return nil
	},
}

func init() {
	Command.AddCommand(
		listCmd,
		currentCmd,
		setCmd,
		showCmd,
		validateCmd,
	)
}
