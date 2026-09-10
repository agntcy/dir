// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package install

import (
	"errors"
	"fmt"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/presenter"
	ctxUtils "github.com/agntcy/dir/cli/util/context"
	"github.com/agntcy/dir/cli/util/prompt"
	"github.com/agntcy/dir/cli/util/reference"
	"github.com/spf13/cobra"
)

// opts is shared by the parent `install`, `run`, and `uninstall` via persistent
// flags on the parent. Only one of those executes per invocation, so a single
// shared set is correct.
var opts options

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
  dirctl install <cid-or-name> --pin      install and hold at this version
  dirctl install uninstall <cid-or-name>  remove what install added
  dirctl install list                     show detected agents and target paths
  dirctl install prune                    drop manifest rows whose artifacts are gone

Every install records what it wrote — record, version, agent, scope, and the
exact files and MCP server keys — in $XDG_CONFIG_HOME/dirctl/installed.json.
A --project install records the repository it wrote into, so one manifest
covers every repository on this machine.

Installing several records at once is a pipe. Filtering belongs to dirctl
search, so install does not carry a second copy of its flags:

  dirctl search --module integration/mcp -o raw | dirctl install --agents all --yes
  dirctl search --skill "code*" -o raw | dirctl install --dry-run

References are read one per line, blanks and # comments ignored. Use
search's -o raw, which is one CID per line; -o jsonl and plain names work
too. The highest version per name wins unless --all-versions is passed. A
piped run cannot prompt, because stdin is the list, so it needs --yes or
--dry-run.

Examples:
  dirctl install cisco.com/agent:v1.0.0
  dirctl install bafyrei... --dry-run
  dirctl install cisco.com/agent --agents claude-code,cursor
  dirctl install uninstall cisco.com/agent
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var input string
		if len(args) > 0 {
			input = args[0]
		}

		switch {
		case input != "":
			return runInstallCmd(cmd, input)
		case hasPipedInput(cmd):
			return runPipedInstall(cmd)
		default:
			return cmd.Help()
		}
	},
}

func init() {
	addSelectionFlags(Command, &opts)
	addPinFlag(Command, &opts)
	addAllVersionsFlag(Command, &opts)

	Command.AddCommand(runCmd)
	Command.AddCommand(uninstallCmd)
	Command.AddCommand(ListCommand)
	Command.AddCommand(PruneCommand)
}

// SkipClientSetup lists the commands root.go must not build a client for.
// Requiring a reachable Directory — or even a configured one — to read local
// state or to remove what the manifest records would be a needless failure.
//
// `uninstall` is here because it reads the manifest and nothing else. That is
// only true now that batch uninstall is gone: expanding search filters was
// the one thing it needed a Directory for.
func SkipClientSetup() []*cobra.Command {
	return []*cobra.Command{ListCommand, PruneCommand, uninstallCmd, UninstallCommand}
}

// selectAgents validates the --agents flag and resolves it to the detected
// agents to act on, printing a note for any explicitly-requested agent that is
// not detected (never installed for undetected agents).
func selectAgents(cmd *cobra.Command, env agentcfg.Env) ([]agentcfg.Agent, error) {
	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return nil, err
	}

	selected, skipped := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen)

	for _, id := range skipped {
		presenter.Printf(cmd, "Skipping %s: not detected.\n", id)
	}

	return selected, nil
}

// pullAndDerive resolves the ref, pulls the record, and derives its artifacts.
// The record itself comes back too, because the manifest row needs the name,
// version, and CID that Artifacts does not carry.
func pullAndDerive(cmd *cobra.Command, input string) (applied, error) {
	c, ok := ctxUtils.GetClientFromContext(cmd.Context())
	if !ok {
		return applied{}, errors.New("failed to get client from context")
	}

	cid, err := reference.ResolveToCID(cmd.Context(), c, input)
	if err != nil {
		return applied{}, fmt.Errorf("resolve reference: %w", err)
	}

	rec, err := c.Pull(cmd.Context(), &corev1.RecordRef{Cid: cid})
	if err != nil {
		return applied{}, fmt.Errorf("failed to pull record: %w", err)
	}

	arts, err := agentinstall.DeriveArtifacts(rec)
	if err != nil {
		return applied{}, err
	}

	return applied{record: rec, arts: arts, pinned: pinRequested(input)}, nil
}

// pinRequested reports whether this install should hold the package at the
// version it resolved to. An explicit `:version` in the reference is the same
// statement as --pin made a different way, so it implies the pin.
func pinRequested(input string) bool {
	return opts.pin || reference.Parse(input).Version != ""
}

// runInstallCmd is the shared body for the parent's bare-positional form and the
// `run` subcommand.
func runInstallCmd(cmd *cobra.Command, input string) error {
	return runApplyCmd(cmd, input, agentinstall.Install, recordInstalls, "\nProceed with these changes?")
}

// runApplyCmd is the single-record flow shared by install and uninstall: pull +
// derive, dry-run plan, confirm, apply, summary, manifest. apply is Install or
// Uninstall, and record is the matching manifest update.
func runApplyCmd(
	cmd *cobra.Command,
	input string,
	apply recordApplyFn,
	record manifestRecordFn,
	confirmPrompt string,
) error {
	item, err := pullAndDerive(cmd, input)
	if err != nil {
		return err
	}

	env := agentcfg.ResolveEnv()
	scope := scopeFromOpts()

	selected, err := selectAgents(cmd, env)
	if err != nil {
		return err
	}

	printScope(cmd)

	plan := apply(env, item.arts, selected, scope, true)
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if len(plan) == 0 {
		return nil
	}

	if !opts.yes && !opts.dryRun {
		ok, err := prompt.Confirm(cmd, confirmPrompt)
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Aborted. No changes made.\n")

			return nil
		}
	}

	item.outcomes = apply(env, item.arts, selected, scope, opts.dryRun)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(item.outcomes, opts.dryRun))

	// A dry run touched nothing, so there is nothing to record.
	if opts.dryRun {
		return nil
	}

	record(cmd, []applied{item}, selected, scope)

	return nil
}
