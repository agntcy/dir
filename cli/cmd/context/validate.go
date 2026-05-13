// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"fmt"
	"strings"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [name]",
	Short: "Validate one or all configured contexts",
	Long: `Validate dirctl client context definitions.

This command validates stored context definitions from the reusable client
config file without applying environment variable overrides. It is intended to
catch missing required fields, unsupported auth modes, and auth-mode-specific
configuration mistakes before a context is used.

If [name] is provided, only that context is validated. If omitted, all
configured contexts are validated in sorted order.

Arguments:

[name] is optional. When omitted, all configured contexts are validated.

Examples:

1. Validate all contexts:
   dirctl context validate

2. Validate one context:
   dirctl context validate prod`,
	Args: cobra.MaximumNArgs(1),
	RunE: runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	contextName := ""
	if len(args) > 0 {
		contextName = args[0]
	}

	results, err := clientconfig.ValidateContexts("", contextName)
	if err != nil {
		return fmt.Errorf("failed to validate contexts: %w", err)
	}

	if len(results) == 0 {
		cmd.Println("No contexts configured.")

		return nil
	}

	var invalid []string

	for _, result := range results {
		if result.Error == nil {
			cmd.Printf("%s: ok\n", result.Name)

			continue
		}

		invalid = append(invalid, result.Name)
		cmd.Printf("%s: invalid: %v\n", result.Name, result.Error)
	}

	if len(invalid) > 0 {
		return fmt.Errorf("invalid context(s): %s", strings.Join(invalid, ", "))
	}

	return nil
}
