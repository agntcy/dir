// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package dirpkg builds the built-in DIR package — the record dirctl compiles
// into itself — and the environment its MCP server entry needs.
//
// It is a package rather than a function inside `dirctl init` because two
// commands need the same derivation, and they must not drift apart.
// `init` installs the DIR skill and MCP server as the last step of onboarding;
// `install upgrade` re-derives them locally instead of pulling the
// Directory-published record of the same name.
//
// The upgrade going through this path is deliberate, and the reason would be
// invisible otherwise. `dirctl mcp serve` reads its target from
// DIRECTORY_CLIENT_* environment variables and from nothing else — neither the
// config file nor current_context — so the installed MCP entry has to carry
// them, which MCPServerEnv is for. The published record's MCP module carries
// no env at all, so installing it would silently repoint the MCP server at the
// default address. Re-deriving also recomputes the overlay from the *current*
// client context rather than replaying whatever was in it when the package was
// installed, and removes any possibility of offering a downgrade when this
// binary leads the server's published build.
package dirpkg

import (
	"fmt"
	"strings"
	"time"

	"github.com/agntcy/dir/cli/internal/agentinstall"
	clientconfig "github.com/agntcy/dir/client/config"
	"github.com/agntcy/dir/server/skill"
)

// The two environment variables the installed MCP entry must always carry.
// auth_mode is as essential as the address: an empty mode makes the client
// attempt OIDC auto-detection, which fails outright when several issuers are
// cached.
const (
	ServerAddressEnv = "DIRECTORY_CLIENT_SERVER_ADDRESS"
	AuthModeEnv      = "DIRECTORY_CLIENT_AUTH_MODE"
)

// The local daemon defaults, shared by the client context `dirctl init` seeds
// and by the MCP env fallback below, so the two cannot disagree about where a
// fresh environment points.
const (
	LocalServerAddress = "localhost:8888"
	LocalAuthMode      = "insecure"
)

// Name is the built-in record's name, which is also the key its manifest rows
// are stored under.
func Name() string { return skill.RecordName }

// Version is the built-in record's version, which for a built-in row is the
// upstream a version check compares against: the upstream is this binary.
func Version() string { return skill.RecordVersion() }

// Artifacts builds the built-in DIR record locally and derives its installable
// artifacts — the skill and the MCP server entry — with the MCP environment
// overlaid. No Directory round-trip.
func Artifacts() (agentinstall.Artifacts, error) {
	rec, err := skill.BuildRecord(time.Now().UTC())
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("build DIR record: %w", err)
	}

	arts, err := agentinstall.DeriveArtifacts(rec)
	if err != nil {
		return agentinstall.Artifacts{}, fmt.Errorf("derive DIR artifacts: %w", err)
	}

	arts.SetMCPEnv(MCPServerEnv())

	return arts, nil
}

// MCPServerEnv builds the DIRECTORY_CLIENT_* environment the spawned
// `dirctl mcp serve` needs to reach the same node dirctl itself uses: the
// current client context's connection settings, with DIRECTORY_CLIENT_*
// overrides applied. Validation is skipped and unknown fields tolerated so a
// partially-set or forward-compatible config still yields usable values; any
// read error degrades to the local default.
//
// Only non-secret fields are projected. The two secrets — auth_token and
// spiffe_token — are deliberately excluded so a bearer token never lands in an
// agent's config file; auth modes that need them still require the user to
// supply the secret through their own environment.
func MCPServerEnv() map[string]string {
	cfg, _, err := clientconfig.Resolve(clientconfig.ResolveOptions{
		SkipValidation:     true,
		AllowUnknownFields: true,
	})
	if err != nil || cfg == nil {
		// No resolvable context (e.g. the user declined init's Step 1): mirror
		// the local default so the server dials the daemon insecurely rather
		// than falling into OIDC auto-detection with an empty auth mode.
		return map[string]string{
			ServerAddressEnv: LocalServerAddress,
			AuthModeEnv:      LocalAuthMode,
		}
	}

	addr := strings.TrimSpace(cfg.ServerAddress)
	if addr == "" {
		addr = LocalServerAddress
	}

	authMode := strings.TrimSpace(cfg.AuthMode)
	if authMode == "" {
		authMode = LocalAuthMode
	}

	env := map[string]string{
		ServerAddressEnv: addr,
		AuthModeEnv:      authMode,
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

	if len(cfg.OIDCScopes) > 0 {
		env["DIRECTORY_CLIENT_OIDC_SCOPES"] = strings.Join(cfg.OIDCScopes, ",")
	}

	if cfg.TlsSkipVerify {
		env["DIRECTORY_CLIENT_TLS_SKIP_VERIFY"] = "true"
	}

	return env
}
