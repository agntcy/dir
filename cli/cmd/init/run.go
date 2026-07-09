// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package init

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	extractor "github.com/agntcy/dir/cli/internal/extractor"
	"github.com/agntcy/dir/cli/presenter"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// isInteractive reports whether the command's stdin is a terminal. Provisioning
// is opt-out for interactive users (Enter provisions) but must not fire an
// ~89 MB download unattended, so a non-TTY run without --yes skips instead.
func isInteractive(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)

	return ok && term.IsTerminal(int(f.Fd())) //nolint:gosec // G115: a file descriptor fits in an int.
}

// printWelcome prints the wizard's opening banner and the list of steps.
func printWelcome(cmd *cobra.Command) {
	presenter.Printf(cmd, `Welcome to dirctl — the CLI for the AGNTCY Directory, a distributed registry
of AI agent records described in OASF.

This wizard sets up your local environment. Steps:
  1. Configure a local client context
  2. Provision the OASF taxonomy extractor
  (more steps — MCP server & skills — coming soon)
`)
}

// run dispatches to the remove flow, or runs the setup steps in order.
func run(cmd *cobra.Command, opts *options) error {
	if opts.remove {
		return runRemove(cmd, opts)
	}

	printWelcome(cmd)

	if err := runContextSetup(cmd, opts); err != nil {
		return err
	}

	return runProvision(cmd, opts)
}

// configFromOpts builds the engine config from flags, resolving defaults.
func configFromOpts(opts *options) extractor.Config {
	return extractor.Config{OASFURL: opts.oasfURL, AssetDir: opts.assetDir}.Resolve()
}

// preferSavedConfig fills cfg's asset dir / OASF URL from the persisted config
// for any value the user did not set via a flag, so a bare re-run targets the
// previously provisioned install instead of the defaults. urlSet/dirSet report
// whether the corresponding flag was explicitly passed — an explicit flag always
// wins (including one set back to the default value). An absent config leaves cfg
// untouched; a present-but-unreadable config is an error (so we fail fast rather
// than provision against defaults and only trip over the bad file later).
func preferSavedConfig(cfg extractor.Config, urlSet, dirSet bool) (extractor.Config, error) {
	saved, err := clientconfig.LoadExtractor("")
	if err != nil {
		return cfg, fmt.Errorf("read extractor config: %w", err)
	}

	if saved == nil {
		return cfg, nil
	}

	if !dirSet && saved.AssetDir != "" {
		cfg.AssetDir = saved.AssetDir
	}

	if !urlSet && saved.OASFURL != "" {
		cfg.OASFURL = saved.OASFURL
	}

	return cfg, nil
}

// resolveConfig builds the effective config from flags, then prefers the saved
// config for any flag the user did not explicitly set.
func resolveConfig(cmd *cobra.Command, opts *options) (extractor.Config, error) {
	return preferSavedConfig(
		configFromOpts(opts),
		cmd.Flags().Changed("oasf-url"),
		cmd.Flags().Changed("asset-dir"),
	)
}

// runProvision provisions the extractor assets, persists the choice, and runs a
// smoke check. It prompts for confirmation and the OASF URL unless --yes.
func runProvision(cmd *cobra.Command, opts *options) error {
	presenter.Printf(cmd, `
Step 2 — OASF taxonomy extractor
Maps free-form text onto the OASF taxonomy so dirctl can enrich records and
(soon) answer free-text searches locally — in-process, with no LLM or API
calls. It needs a one-time ~89 MB model + the taxonomy downloaded to your
machine; provisioned once, reused everywhere.
`)

	cfg, err := resolveConfig(cmd, opts)
	if err != nil {
		return err
	}

	if extractor.IsProvisioned(cfg) {
		presenter.Printf(cmd, "Extractor already provisioned at %s (OASF URL %s).\n",
			cfg.AssetDir, cfg.OASFURL)
		presenter.Printf(cmd, "Re-running updates it; use --remove to tear it down.\n")
	}

	if !opts.yes {
		// Opt-out, but never unattended: a non-interactive run without --yes
		// skips rather than triggering an ~89 MB download in CI/scripts.
		if !isInteractive(cmd) {
			presenter.Printf(cmd, "Skipping extractor provisioning (non-interactive). Pass --yes to provision.\n")

			return nil
		}

		ok, err := confirm(cmd, "\nProvision the OASF extractor (~89 MB)?", true)
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Skipped. No changes made.\n")

			return nil
		}

		url, err := promptOASFURL(cmd, cfg.OASFURL)
		if err != nil {
			return err
		}

		cfg.OASFURL = url
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	presenter.Printf(cmd, "\nProvisioning extractor assets to %s\n", cfg.AssetDir)

	captured, err := runWithSpinner(cmd.Context(), os.Stdout, phaseDownload, phaseFor,
		func(ctx context.Context) error { return extractor.Provision(ctx, cfg) })
	if err != nil {
		// cobra prints the returned error ("Error: …"); we add the captured
		// provisioning output beneath it so failures are diagnosable.
		printDetails(cmd, captured)

		return err
	}

	// Persist the choice before the smoke check so a diagnostic smoke-check
	// failure never discards the on-disk assets or the saved config.
	if err := clientconfig.SaveExtractor("", &clientconfig.Extractor{
		OASFURL:  cfg.OASFURL,
		AssetDir: cfg.AssetDir,
	}); err != nil {
		return fmt.Errorf("save extractor config: %w", err)
	}

	captured, err = runWithSpinner(cmd.Context(), os.Stdout, "Verifying setup…", nil,
		func(ctx context.Context) error { return extractor.SmokeCheck(ctx, cfg) })
	if err != nil {
		// The assets and saved config are kept (persisted above) so a retry is
		// cheap, but verification failed — return a non-nil error so `--yes`/CI
		// runs surface the broken setup instead of reporting success. cobra
		// prints the error; we add context and the captured details.
		presenter.Printf(cmd, "⚠ Assets provisioned at %s, but verification failed. Re-run `dirctl init` to retry.\n", cfg.AssetDir)
		printDetails(cmd, captured)

		return fmt.Errorf("extractor verification failed: %w", err)
	}

	presenter.Printf(cmd, "✔ Extractor ready.\n  asset dir: %s\n  OASF URL:  %s\nEnrichment and free-text search can now run locally.\n",
		cfg.AssetDir, cfg.OASFURL)

	return nil
}

// printDetails surfaces captured provisioning output as an indented block, used
// only when a step fails so the raw logs stay hidden on the happy path.
func printDetails(cmd *cobra.Command, captured string) {
	if details := strings.TrimSpace(captured); details != "" {
		presenter.Errorf(cmd, "\nDetails:\n%s\n", indentLines(details, "  "))
	}
}

// runRemove tears down the provisioned assets and clears the saved config.
func runRemove(cmd *cobra.Command, opts *options) error {
	// Prefer the persisted asset dir so removal targets what was provisioned.
	cfg, err := resolveConfig(cmd, opts)
	if err != nil {
		return err
	}

	if !opts.yes {
		ok, err := confirm(cmd, fmt.Sprintf("Remove extractor assets at %s?", cfg.AssetDir), false)
		if err != nil {
			return err
		}

		if !ok {
			presenter.Printf(cmd, "Aborted. Nothing removed.\n")

			return nil
		}
	}

	if err := extractor.Teardown(cfg); err != nil {
		return err
	}

	if err := clientconfig.ClearExtractor(""); err != nil {
		return fmt.Errorf("clear extractor config: %w", err)
	}

	presenter.Printf(cmd, "Removed extractor assets at %s and cleared saved config.\n", cfg.AssetDir)

	return nil
}

// promptOASFURL asks for the OASF URL, returning def when the user enters
// nothing (or stdin is at EOF).
//
//nolint:unparam
func promptOASFURL(cmd *cobra.Command, def string) (string, error) {
	presenter.Printf(cmd, "OASF URL [%s]: ", def)

	reader := bufio.NewReader(cmd.InOrStdin())

	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return def, nil
	}

	answer := strings.TrimSpace(line)
	if answer == "" {
		return def, nil
	}

	return answer, nil
}
