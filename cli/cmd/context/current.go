// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package context

import (
	"encoding/json"
	"errors"
	"fmt"

	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active context",
	Long: `Show the active dirctl client context.

This command prints the persisted current_context from the dirctl client config
file. It intentionally ignores one-off context overrides such as --context,
and DIRECTORY_CLIENT_CONTEXT so it stays consistent with dirctl context list,
which marks the persisted current_context.

By default, it prints only the persisted context name followed by a newline.
Use --quiet for shell prompts where missing context should produce no output
and no error. Use --json for stable machine-readable output.

Examples:

1. Show the active context:
   dirctl context current

2. Switch the persisted current context:
   dirctl context set staging

3. Use in a shell prompt helper:
   dirctl_context_prompt() {
     ctx="$(dirctl context current --quiet 2>/dev/null)" || return
     [ -n "$ctx" ] && printf "[dir:%s] " "$ctx"
   }

4. Get machine-readable status:
   dirctl context current --json`,
	Args: cobra.NoArgs,
	RunE: runCurrent,
}

type currentContextOutput struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
}

func init() {
	addCurrentFlags(currentCmd)
}

func addCurrentFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("quiet", false, "Print only the persisted current context name; print nothing when no context is set")
	cmd.Flags().Bool("json", false, "Print persisted current context details as JSON")
}

func runCurrent(cmd *cobra.Command, _ []string) error {
	quiet, err := boolFlag(cmd, "quiet")
	if err != nil {
		return err
	}

	jsonOutput, err := boolFlag(cmd, "json")
	if err != nil {
		return err
	}

	if quiet && jsonOutput {
		return errors.New("use only one of --quiet or --json")
	}

	current, err := clientconfig.CurrentContext("")
	if err != nil {
		return fmt.Errorf("failed to get current context: %w", err)
	}

	if current.Name == "" {
		if quiet {
			return nil
		}

		if jsonOutput {
			return printCurrentContextJSON(cmd, current)
		}

		return errors.New("no current context is set")
	}

	if jsonOutput {
		return printCurrentContextJSON(cmd, current)
	}

	if quiet {
		cmd.Print(current.Name)

		return nil
	}

	cmd.Println(current.Name)

	return nil
}

func boolFlag(cmd *cobra.Command, name string) (bool, error) {
	if cmd.Flags().Lookup(name) == nil {
		return false, nil
	}

	value, err := cmd.Flags().GetBool(name)
	if err != nil {
		return false, fmt.Errorf("failed to read %s flag: %w", name, err)
	}

	return value, nil
}

func printCurrentContextJSON(cmd *cobra.Command, current *clientconfig.ResolvedContext) error {
	output := currentContextOutput{
		Name:   current.Name,
		Source: current.Source,
		Path:   current.Path,
	}

	encoder := json.NewEncoder(cmd.OutOrStdout())
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode current context output: %w", err)
	}

	return nil
}
