// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"errors"
	"fmt"

	"github.com/agntcy/dir/cli/config"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active context",
	Long: `Show the active dirctl client context.

This command prints the context selected for the current invocation. Selection
uses the same context precedence as client config resolution:

1. --context
2. DIRCTL_CONTEXT
3. DIRECTORY_CLIENT_CONTEXT
4. current_context from the config file

It prints only the selected context name, which keeps the output suitable for
shell prompts and scripts.

Examples:

1. Show the active context:
   dirctl context current

2. Show what an explicit override selects:
   dirctl --context staging context current`,
	Args: cobra.NoArgs,
	RunE: runCurrent,
}

func runCurrent(cmd *cobra.Command, _ []string) error {
	current, err := clientconfig.CurrentContext("", config.Context)
	if err != nil {
		return fmt.Errorf("failed to get current context: %w", err)
	}

	if current.Name == "" {
		return errors.New("no current context is set")
	}

	cmd.Println(current.Name)

	return nil
}
