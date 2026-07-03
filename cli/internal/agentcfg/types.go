// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package agentcfg is the shared placement engine for writing artifacts (MCP
// server entries and Agent Skills) into the configuration files of installed AI
// coding agents. Callers supply the artifact identity and content; this package
// owns the per-agent paths, JSON/YAML/TOML codec, managed-block merge, atomic
// writes, detection, and outcome reporting. Used by `dirctl install` and
// `dirctl init`.
package agentcfg

import (
	"github.com/agntcy/dir/cli/internal/agentcfg/codec"
)

// Env carries the ambient values descriptors and engines need, injected so
// resolvers and tests stay pure (no direct os calls inside descriptors).
type Env struct {
	Home string
	GOOS string
	Cwd  string
}

// EntryStyle selects how an MCP server entry value is shaped for a given agent.
type EntryStyle int

const (
	// CommandArgsEnv is the common {command, args, env} entry shape.
	CommandArgsEnv EntryStyle = iota
	// ZedContextServer is Zed's context_servers custom shape ({source, command, args, env}).
	ZedContextServer
)

// MCPTarget describes where and how to install the MCP server entry for an agent.
type MCPTarget struct {
	// ConfigPath resolves the global config file path for the given environment.
	ConfigPath func(env Env) (string, error)
	// Format is the config file encoding.
	Format codec.Format
	// ServersKey is the nested key path to the servers map (e.g. ["mcpServers"]).
	ServersKey []string
	// EntryStyle shapes the entry value.
	EntryStyle EntryStyle
}

// SkillStrategy selects how the DIR skill/rules are rendered into an agent.
type SkillStrategy int

const (
	// SkillFolder writes SKILL.md verbatim into an agent's skills directory.
	SkillFolder SkillStrategy = iota
	// DedicatedFile writes a single rules file owned by us, with tool-specific frontmatter.
	DedicatedFile
	// ManagedBlock inserts/replaces a delimited block inside a shared instruction file.
	ManagedBlock
)

// Renderer turns the canonical SKILL.md into the on-disk bytes for a target.
// Implemented in the skill subsystem (Phase 2).
type Renderer func(canonical string) ([]byte, error)

// SkillTarget describes where and how to install the DIR skill/rules for an agent.
type SkillTarget struct {
	Strategy SkillStrategy
	// Path resolves the global target from the env and per-record slug. If it
	// returns ErrNoGlobalPath, the engine retries with ProjectPath.
	Path func(env Env, slug string) (string, error)
	// ProjectPath resolves a per-repo (cwd) fallback path from env and slug.
	ProjectPath func(env Env, slug string) (string, error)
	// Render produces the on-disk bytes (DedicatedFile/ManagedBlock). Nil for SkillFolder.
	Render Renderer
	// SharedWith names another agent ID that resolves to the same file, so the
	// shared artifact is installed only once.
	SharedWith string
}

// Agent is a data descriptor for one supported AI coding agent.
type Agent struct {
	ID     string // stable identifier, e.g. "claude-code"
	Name   string // human-readable name, e.g. "Claude Code"
	Flag   string // CLI selection flag, e.g. "claude-code"
	Detect func(env Env) bool

	MCP   *MCPTarget   // nil if we target no MCP config for this agent
	Skill *SkillTarget // nil if we install no skill for this agent
}
