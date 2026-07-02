// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package install

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

// opts and agentFlags are shared by the parent `install`, `run`, and `uninstall`
// via persistent flags on the parent. Only one of those executes per invocation,
// so a single shared set is correct.
var (
	opts       options
	agentFlags map[string]*bool
)

// Command is the `dirctl install` parent. With a positional CID/name it runs an
// install (equivalent to `install run`); with no argument it prints help.
var Command = &cobra.Command{
	Use:   "install <cid-or-name[:version][@digest]>",
	Short: "Install a record's artifacts into detected AI coding agents",
	Long: `Pull a record from the active Directory and install its artifacts — an MCP
server entry and/or an Agent Skill, derived from the record's OASF modules —
directly into the configuration of detected AI coding agents.

  dirctl install <cid-or-name>            detect agents, preview, confirm, install
  dirctl install run <cid-or-name>        same as above
  dirctl install uninstall <cid-or-name>  remove what install added
  dirctl install list                     show detected agents and target paths

Examples:
  dirctl install my-agent:1.0.0
  dirctl install bafyrei... --dry-run
  dirctl install my-agent --mcp --claude-code --cursor
  dirctl install uninstall my-agent --all
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		return runInstallCmd(cmd, args[0])
	},
}

func init() {
	agentFlags = addSelectionFlags(Command, &opts)

	Command.AddCommand(runCmd)
	Command.AddCommand(uninstallCmd)
	Command.AddCommand(ListCommand)
}

// pullAndDerive resolves the ref, pulls the record, and derives its artifacts.
func pullAndDerive(cmd *cobra.Command, input string) (artifacts, error) {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return artifacts{}, errors.New("failed to get client from context")
	}

	cid, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return artifacts{}, fmt.Errorf("resolve reference: %w", err)
	}

	record, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: cid})
	if err != nil {
		return artifacts{}, fmt.Errorf("failed to pull record: %w", err)
	}

	return deriveArtifacts(record)
}

// runInstallCmd is the shared body for the parent's bare-positional form and the
// `run` subcommand: pull + derive, dry-run plan, confirm, apply, summary.
func runInstallCmd(cmd *cobra.Command, input string) error {
	arts, err := pullAndDerive(cmd, input)
	if err != nil {
		return err
	}

	env := agentcfg.ResolveEnv()
	set := agentcfg.ResolveArtifacts(opts.mcpOnly, opts.skillOnly)
	chosen := chosenFrom(agentFlags)
	sels := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen, opts.all, opts.force)

	plan := runInstall(env, arts, sels, set, true)
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if len(plan) == 0 {
		return nil
	}

	if !opts.yes && !opts.dryRun {
		ok, err := confirm(cmd, "\nProceed with these changes?")
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Aborted. No changes made.\n")

			return nil
		}
	}

	outcomes := runInstall(env, arts, sels, set, opts.dryRun)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, opts.dryRun))

	return nil
}

// confirm prompts for a yes/no answer on the command's input stream.
func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	presenter.Printf(cmd, "%s [y/N]: ", prompt)

	reader := bufio.NewReader(cmd.InOrStdin())

	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	answer := strings.ToLower(strings.TrimSpace(line))

	return answer == "y" || answer == "yes", nil
}
