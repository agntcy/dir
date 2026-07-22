// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck
package init

import (
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/agentinstall"
	"github.com/agntcy/dir/cli/presenter"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/agntcy/dir/server/skill"
	"github.com/spf13/cobra"
)

// `dirctl mcp serve` reads its Directory connection settings only from
// DIRECTORY_CLIENT_* env — it consults neither the client config file nor
// current_context. So the installed MCP entry must carry them. auth_mode is as
// essential as the address: an empty mode makes the client attempt OIDC
// auto-detection, which fails outright when multiple issuers are cached.
const (
	dirServerAddressEnv = "DIRECTORY_CLIENT_SERVER_ADDRESS"
	dirAuthModeEnv      = "DIRECTORY_CLIENT_AUTH_MODE"
)

const agentStepIntro = `
Step 3 — Directory MCP server & skills
Wire this Directory into your AI coding agents: an MCP server entry (so an agent
can push, search, and pull records) plus the DIR skill (a usage guide). Content
is built in — no Directory connection is made. Writes are idempotent and atomic.
`

// agentSelector picks a subset of the candidate agents for one artifact. It is
// injected so the interactive checkbox selector (promptMultiSelect, production)
// can be swapped for a deterministic fake in tests. Returning an empty slice
// means "install nowhere".
type agentSelector func(cmd *cobra.Command, title string, candidates []agentcfg.Agent) ([]agentcfg.Agent, error)

// interactiveCheck reports whether the command may prompt. It is a package var
// (defaulting to isInteractive) so tests can drive the interactive branch without
// a real TTY.
var interactiveCheck = isInteractive

// dirArtifacts builds the built-in DIR record locally and derives its
// installable artifacts (skill + MCP server). No Directory round-trip.
func dirArtifacts() (agentinstall.Artifacts, error) {
	rec, err := skill.BuildRecord(time.Now().UTC())
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("build DIR record: %w", err)
	}

	arts, err := agentinstall.DeriveArtifacts(rec)
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("derive DIR artifacts: %w", err)
	}

	return arts, nil
}

// mcpServerEnv builds the DIRECTORY_CLIENT_* env the spawned `dirctl mcp serve`
// needs to reach the same node dirctl itself uses: the current client context's
// connection settings, with DIRECTORY_CLIENT_* env overrides applied. Validation
// is skipped and unknown fields tolerated so a partially-set or forward-compat
// config still yields usable values; any read error degrades to the local
// default (insecure daemon).
//
// Only non-secret fields are projected. The two secrets — auth_token and
// spiffe_token — are deliberately excluded so a bearer token never lands in an
// agent's config file; auth modes that need them still require the user to
// supply the secret via their own environment.
func mcpServerEnv() map[string]string {
	cfg, _, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		SkipValidation:     true,
		AllowUnknownFields: true,
	})
	if err != nil || cfg == nil {
		// No resolvable context (e.g. the user declined Step 1): mirror the
		// local default so the server dials the daemon insecurely rather than
		// falling into OIDC auto-detection with an empty auth mode.
		return map[string]string{
			dirServerAddressEnv: localServerAddress,
			dirAuthModeEnv:      localAuthMode,
		}
	}

	addr := strings.TrimSpace(cfg.ServerAddress)
	if addr == "" {
		addr = localServerAddress
	}

	authMode := strings.TrimSpace(cfg.AuthMode)
	if authMode == "" {
		authMode = localAuthMode
	}

	env := map[string]string{
		dirServerAddressEnv: addr,
		dirAuthModeEnv:      authMode,
	}

	// Non-secret, mode-specific fields — projected only when set.
	for k, v := range map[string]string{
		"DIRECTORY_CLIENT_TLS_CERT_FILE":      cfg.TlsCertFile,
		"DIRECTORY_CLIENT_TLS_KEY_FILE":       cfg.TlsKeyFile,
		"DIRECTORY_CLIENT_TLS_CA_FILE":        cfg.TlsCAFile,
		"DIRECTORY_CLIENT_SPIFFE_SOCKET_PATH": cfg.SpiffeSocketPath,
		"DIRECTORY_CLIENT_JWT_AUDIENCE":       cfg.JWTAudience,
		"DIRECTORY_CLIENT_OIDC_ISSUER":        cfg.OIDCIssuer,
		"DIRECTORY_CLIENT_OIDC_CLIENT_ID":     cfg.OIDCClientID,
	} {
		if s := strings.TrimSpace(v); s != "" {
			env[k] = s
		}
	}

	if cfg.TlsSkipVerify {
		env["DIRECTORY_CLIENT_TLS_SKIP_VERIFY"] = "true"
	}

	return env
}

// runAgentSetup runs Step 3 against the resolved ambient environment, using the
// interactive checkbox prompt for per-agent selection.
func runAgentSetup(cmd *cobra.Command, opts *options) error {
	return installAgents(cmd, agentcfg.ResolveEnv(), opts, promptMultiSelect)
}

// installAgents is the testable core of Step 3: detect the candidate agents,
// then (interactively) ask which of them get the DIR skill and, separately,
// which get the DIR MCP server — installing each artifact via the shared
// agentinstall engine. selectAgents is the per-artifact chooser (injected).
func installAgents(cmd *cobra.Command, env agentcfg.Env, opts *options, selectAgents agentSelector) error {
	presenter.Printf(cmd, "%s", agentStepIntro)

	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err
	}

	candidates, skipped := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen)
	for _, id := range skipped {
		presenter.Printf(cmd, "Skipping %s: not detected.\n", id)
	}

	if len(candidates) == 0 {
		presenter.Printf(cmd, "No supported AI coding agents detected; skipping.\n")

		return nil
	}

	arts, err := dirArtifacts()
	if err != nil {
		return err
	}

	// Point the built-in DIR MCP server at the resolved context. `dirctl mcp
	// serve` takes its target only from DIRECTORY_CLIENT_* env, so without this
	// the spawned server ignores the context just configured and falls back to
	// the default address.
	arts.SetMCPEnv(mcpServerEnv())

	// Non-interactive: never prompt. With --yes, install both artifacts into
	// every candidate; without it, skip rather than act unattended.
	if opts.yes || !interactiveCheck(cmd) {
		if !opts.yes {
			presenter.Printf(cmd, "Skipping MCP server & skill setup (non-interactive). Pass --yes to install.\n")

			return nil
		}

		apply(cmd, env, arts.SkillOnly(), candidates, "DIR skill")
		apply(cmd, env, arts.MCPOnly(), candidates, "DIR MCP server")

		return nil
	}

	// Interactive: one prompt per artifact, each pre-selecting all candidates.
	skillAgents, err := selectAgents(cmd, "Install the DIR skill into:", candidates)
	if err != nil {
		return err
	}

	apply(cmd, env, arts.SkillOnly(), skillAgents, "DIR skill")

	mcpAgents, err := selectAgents(cmd, "Install the DIR MCP server into:", candidates)
	if err != nil {
		return err
	}

	apply(cmd, env, arts.MCPOnly(), mcpAgents, "DIR MCP server")

	return nil
}

// apply installs one artifact set into the chosen agents and prints the summary.
// An empty selection is a no-op with a short note (the user deselected everyone).
func apply(cmd *cobra.Command, env agentcfg.Env, arts agentinstall.Artifacts, agents []agentcfg.Agent, label string) {
	if len(agents) == 0 {
		presenter.Printf(cmd, "%s: no agents selected; skipping.\n", label)

		return
	}

	outcomes := agentinstall.Install(env, arts, agents, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))
}

// removeAgents strips the built-in DIR MCP server & skill from detected agents.
// It mirrors installAgents' selection and TTY/--yes gating.
func removeAgents(cmd *cobra.Command, env agentcfg.Env, opts *options) error {
	chosen, err := agentcfg.ParseSelection(opts.agents)
	if err != nil {
		return err
	}

	selected, _ := agentcfg.ResolveSelection(agentcfg.Registry(), env, chosen)
	if len(selected) == 0 {
		return nil
	}

	arts, err := dirArtifacts()
	if err != nil {
		return err
	}

	plan := agentinstall.Uninstall(env, arts, selected, true)
	if len(plan) == 0 {
		return nil
	}

	presenter.Printf(cmd, "\nThe DIR MCP server & skill will be removed from:\n")
	presenter.Printf(cmd, "%s", agentcfg.FormatPlan(plan))

	if !opts.yes {
		// Defensive, and mirrors installAgents' gating: in the wizard, runRemove
		// already confirms before calling us, so a non-interactive run aborts
		// there first. This guard only matters if removeAgents is ever called
		// standalone — it must never act unattended.
		if !isInteractive(cmd) {
			return nil
		}

		ok, err := confirm(cmd, "Remove the DIR MCP server & skill from these agents?", false)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}
	}

	outcomes := agentinstall.Uninstall(env, arts, selected, false)
	presenter.Printf(cmd, "%s", agentcfg.FormatSummary(outcomes, false))

	return nil
}
