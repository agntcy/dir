// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package init implements `dirctl init` — the onboarding step that provisions
// the OASF taxonomy extractor's local assets and records the choice in the
// dirctl config. Additional wizard steps (MCP server / skills) are layered on
// by #1705.
package init

import (
	"bufio"
	"strings"

	"github.com/agntcy/dir/cli/presenter"
	"github.com/spf13/cobra"
)

var opts options

// Command is the `dirctl init` command.
var Command = &cobra.Command{
	Use:   "init",
	Short: "Set up the local environment to work with the Directory",
	Long: `Provision the OASF taxonomy extractor's local assets (a ~89 MB
sentence-transformer model plus the OASF taxonomy) so record enrichment and
free-text search can run locally, in-process, with no LLM dependency. The chosen
OASF endpoint and asset directory are saved to the dirctl config for later use.

  dirctl init                 provision (prompts for confirmation + OASF URL)
  dirctl init --yes           provision non-interactively with defaults
  dirctl init --remove        remove provisioned assets and clear saved config
`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		return run(cmd, &opts)
	},
}

func init() {
	addFlags(Command, &opts)
}

// confirm prompts for a yes/no answer on the command's input stream. defaultYes
// selects the answer for an empty line (just Enter) and the displayed hint
// ([Y/n] vs [y/N]). EOF with no input (a non-interactive stdin) always returns
// false, so a heavy or destructive default is never assumed without a keystroke.
//
//nolint:unparam // error return is part of the prompt contract used by callers.
func confirm(cmd *cobra.Command, prompt string, defaultYes bool) (bool, error) {
	hint := "[y/N]"
	if defaultYes {
		hint = "[Y/n]"
	}

	presenter.Printf(cmd, "%s %s: ", prompt, hint)

	reader := bufio.NewReader(cmd.InOrStdin())

	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, nil
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default: // empty line (Enter) or anything else falls back to the default
		return defaultYes, nil
	}
}
