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

// Present reports whether anything the row names is still installed.
//
// One surviving artifact is enough. A record can yield both a skill and an MCP
// entry, and a user who deletes only the skill folder still has half of the
// package wired in — a state an upgrade can fix, so calling the row missing
// would be wrong.
//
// A row that names no artifacts is reported present: there is nothing to
// contradict it, and claiming otherwise would strand the row.
func Present(entry pkgstate.Entry, env agentcfg.Env) bool {
	found := Inspect(entry, env)
	if len(found) == 0 {
		return true
	}

	for _, a := range found {
		if a.Present {
			return true
		}
	}

	return false
}

func inspectSkill(entry pkgstate.Entry) Installed {
	skill := Installed{
		Kind:    agentcfg.ArtifactSkill,
		Path:    entry.SkillPath,
		Present: exists(entry.SkillPath),
	}

	for _, name := range entry.SkillFiles {
		skill.Files = append(skill.Files, InstalledFile{
			Name:    name,
			Present: exists(filepath.Join(entry.SkillPath, name)),
		})
	}

	return skill
}

// inspectMCP checks each recorded server key against the agent's config file.
// An agent this binary does not know, or one with no MCP location, leaves the
// entry unlocated: it is reported present, because nothing was checked and a
// row must not be called missing on the strength of a lookup that never ran.
func inspectMCP(entry pkgstate.Entry, env agentcfg.Env) []Installed {
	agent, known := agentcfg.ByID(entry.Agent)
	scope, env := placement(entry, env)

	servers := make([]Installed, 0, len(entry.MCPServers))

	for _, server := range entry.MCPServers {
		mcp := Installed{Kind: agentcfg.ArtifactMCP, Server: server}

		if !known || agent.MCP == nil {
			mcp.Present = true
			servers = append(servers, mcp)

			continue
		}

		mcp.Path = agentcfg.ResolveMCPPath(agent.MCP, env, scope)
		mcp.Present = agentcfg.MCPEntryPresent(agent.MCP, env, server, scope)

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

func exists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
