// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentinstall

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/internal/agentcfg"
	"github.com/agntcy/dir/cli/internal/pkgstate"
)

// Installed is one artifact a manifest row names, together with whether it is
// still where the row says it is.
//
// The manifest is the only provenance an installed artifact has, so anything
// reporting on a row has to look at what the row claims rather than trust it:
// a user can move or delete a skill folder, or edit an MCP entry out of an
// agent's config, and nothing tells dirctl about it.
type Installed struct {
	// Kind is agentcfg.ArtifactSkill or agentcfg.ArtifactMCP.
	Kind string `json:"kind"`

	// Path is the skill file or folder, or the agent config file the MCP entry
	// lives in. Empty when this dirctl cannot resolve a location for the row's
	// agent.
	Path string `json:"path,omitempty"`

	// Server is the config key an MCP entry is stored under. Empty for a skill.
	Server string `json:"server,omitempty"`

	// Files are a skill bundle's files, relative to Path. Empty for a
	// single-file skill, where Path is itself the file.
	Files []InstalledFile `json:"files,omitempty"`

	// Present is whether the artifact was found.
	Present bool `json:"present"`

	// Unchecked marks an artifact whose presence could not be established at
	// all: an unreadable skill path, an agent config that cannot be read or
	// parsed, or an agent this binary does not know. Such an artifact is
	// neither present nor confirmed gone, and nothing may be deleted on the
	// strength of it.
	Unchecked bool `json:"unchecked,omitempty"`
}

// InstalledFile is one file of a skill bundle.
type InstalledFile struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
}

// Inspect lists the artifacts a manifest row names and checks each one, in a
// fixed order: the skill first, then one entry per MCP server key.
func Inspect(entry pkgstate.Entry, env agentcfg.Env) []Installed {
	var found []Installed

	if entry.SkillPath != "" {
		found = append(found, inspectSkill(entry))
	}

	if len(entry.MCPServers) > 0 {
		found = append(found, inspectMCP(entry, env)...)
	}

	return found
}

// Present reports whether the row still stands for something installed.
//
// It is the negation of "confirmed gone", not "confirmed there", and the
// difference matters because callers delete manifest rows on the strength of
// it. A row counts as gone only when **every** artifact it names was looked at
// and found absent. Three cases therefore keep it present:
//
//   - One surviving artifact. A record can yield both a skill and an MCP
//     entry, and a user who deleted only the skill folder still has half the
//     package wired in — a state an upgrade can fix.
//   - An artifact that could not be checked: an unreadable path, a malformed
//     agent config, an agent this binary does not know. Dropping the row there
//     would destroy the only provenance for something that may well exist.
//   - A row that names no artifacts. There is nothing to contradict it.
func Present(entry pkgstate.Entry, env agentcfg.Env) bool {
	for _, a := range Inspect(entry, env) {
		if a.Present || a.Unchecked {
			return true
		}
	}

	return len(Inspect(entry, env)) == 0
}

func inspectSkill(entry pkgstate.Entry) Installed {
	present, checked := exists(entry.SkillPath)

	skill := Installed{
		Kind:      agentcfg.ArtifactSkill,
		Path:      entry.SkillPath,
		Present:   present,
		Unchecked: !checked,
	}

	for _, name := range entry.SkillFiles {
		filePresent, _ := exists(filepath.Join(entry.SkillPath, name))
		skill.Files = append(skill.Files, InstalledFile{Name: name, Present: filePresent})
	}

	return skill
}

// inspectMCP checks each recorded server key against the agent's config file.
// An agent this binary does not know, or one with no MCP location, leaves the
// entry unchecked rather than absent: nothing was looked at, and a row must
// not be called gone on the strength of a lookup that never ran.
func inspectMCP(entry pkgstate.Entry, env agentcfg.Env) []Installed {
	agent, known := agentcfg.ByID(entry.Agent)
	scope, env := placement(entry, env)

	servers := make([]Installed, 0, len(entry.MCPServers))

	for _, server := range entry.MCPServers {
		mcp := Installed{Kind: agentcfg.ArtifactMCP, Server: server}

		if !known || agent.MCP == nil {
			mcp.Unchecked = true
			servers = append(servers, mcp)

			continue
		}

		mcp.Path = agentcfg.ResolveMCPPath(agent.MCP, env, scope)

		present, checked := agentcfg.MCPEntryChecked(agent.MCP, env, server, scope)
		mcp.Present = present
		mcp.Unchecked = !checked

		servers = append(servers, mcp)
	}

	return servers
}

// placement maps a row onto the placement engine's scope and the environment
// that resolves its paths.
//
// A project row names the repository it was written into, and the placement
// engine resolves project paths from Env.Cwd, so the row's own directory is
// substituted for the caller's. Without that, reporting on or removing a row
// would look in whatever directory the command happens to be run from and find
// nothing — which is also what lets a project install be listed, checked, and
// removed from anywhere.
func placement(entry pkgstate.Entry, env agentcfg.Env) (agentcfg.Scope, agentcfg.Env) {
	if entry.Scope.IsGlobal() {
		return agentcfg.Global, env
	}

	env.Cwd = entry.Scope.Dir()

	return agentcfg.Project, env
}

// exists reports whether path is there, and — as its second result — whether
// the question could be answered at all. A permission error or a broken mount
// is not an absence.
func exists(path string) (bool, bool) {
	if _, err := os.Stat(path); err == nil {
		return true, true
	} else if os.IsNotExist(err) {
		return false, true
	}

	return false, false
}
