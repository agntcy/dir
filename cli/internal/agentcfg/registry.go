// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"os"
	"path/filepath"

	"github.com/agntcy/dir/cli/internal/agentcfg/codec"
)

// Registry returns the descriptors for every supported AI coding agent.
// Adding an agent is a matter of appending a descriptor here.
func Registry() []Agent {
	return []Agent{
		{
			ID:     "claude-code",
			Name:   "Claude Code",
			Flag:   "claude-code",
			Detect: detectByMarker(claudeCodeMarker),
			MCP:    jsonMCP(claudeCodeMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy: SkillFolder,
				Path:     claudeCodeSkillPath,
			},
		},
		{
			ID:     "claude-desktop",
			Name:   "Claude Desktop",
			Flag:   "claude-desktop",
			Detect: detectByMarker(claudeDesktopMarker),
			MCP:    jsonMCP(claudeDesktopMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy:   SkillFolder,
				Path:       claudeCodeSkillPath, // shares Claude Code's skills folder
				SharedWith: "claude-code",
			},
		},
		{
			ID:     "cursor",
			Name:   "Cursor",
			Flag:   "cursor",
			Detect: detectByMarker(cursorMarker),
			MCP:    jsonMCP(cursorMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy:    DedicatedFile,
				Path:        cursorNoGlobalSkill, // no global rules mechanism
				ProjectPath: cursorProjectSkillPath,
				Render:      renderCursor,
			},
		},
		{
			ID:     "vscode",
			Name:   "VS Code (Copilot)",
			Flag:   "vscode",
			Detect: detectByMarker(vscodeMarker),
			MCP:    jsonMCP(vscodeMCPPath, "servers"),
			Skill: &SkillTarget{
				Strategy: DedicatedFile,
				Path:     copilotSkillPath,
				Render:   renderCopilot,
			},
		},
		{
			ID:     "windsurf",
			Name:   "Windsurf",
			Flag:   "windsurf",
			Detect: detectByMarker(windsurfMarker),
			MCP:    jsonMCP(windsurfMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy: ManagedBlock,
				Path:     windsurfSkillPath,
				Render:   renderManagedInner,
			},
		},
		{
			ID:     "cline",
			Name:   "Cline",
			Flag:   "cline",
			Detect: detectByMarker(clineMarker),
			MCP:    jsonMCP(clineMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy: DedicatedFile,
				Path:     clineSkillPath,
				Render:   renderCline,
			},
		},
		{
			ID:     "roo",
			Name:   "Roo Code",
			Flag:   "roo",
			Detect: detectByMarker(rooMarker),
			MCP:    jsonMCP(rooMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy: DedicatedFile,
				Path:     rooSkillPath,
				Render:   renderRoo,
			},
		},
		{
			ID:     "gemini",
			Name:   "Gemini CLI",
			Flag:   "gemini",
			Detect: detectByMarker(geminiMarker),
			MCP:    jsonMCP(geminiMCPPath, "mcpServers"),
			Skill: &SkillTarget{
				Strategy: ManagedBlock,
				Path:     geminiSkillPath,
				Render:   renderManagedInner,
			},
		},
		{
			ID:     "opencode",
			Name:   "OpenCode",
			Flag:   "opencode",
			Detect: detectByMarker(opencodeMarker),
			MCP:    jsonMCP(opencodeMCPPath, "mcp"),
			Skill: &SkillTarget{
				Strategy: ManagedBlock,
				Path:     opencodeSkillPath,
				Render:   renderManagedInner,
			},
		},
		{
			ID:     "zed",
			Name:   "Zed",
			Flag:   "zed",
			Detect: detectByMarker(zedMarker),
			MCP: &MCPTarget{
				ConfigPath: zedMCPPath,
				Format:     codec.JSON,
				ServersKey: []string{"context_servers"},
				EntryStyle: ZedContextServer,
			},
			Skill: &SkillTarget{
				Strategy: SkillFolder,
				Path:     zedSkillPath,
			},
		},
		{
			ID:     "continue",
			Name:   "Continue",
			Flag:   "continue",
			Detect: detectByMarker(continueMarker),
			MCP: &MCPTarget{
				ConfigPath: continueMCPPath,
				Format:     codec.YAML,
				ServersKey: []string{"mcpServers"},
				EntryStyle: CommandArgsEnv,
			},
			Skill: &SkillTarget{
				Strategy: DedicatedFile,
				Path:     continueSkillPath,
				Render:   renderContinue,
			},
		},
		{
			ID:     "codex",
			Name:   "Codex CLI",
			Flag:   "codex",
			Detect: detectByMarker(codexMarker),
			MCP: &MCPTarget{
				ConfigPath: codexMCPPath,
				Format:     codec.TOML,
				ServersKey: []string{"mcp_servers"},
				EntryStyle: CommandArgsEnv,
			},
			Skill: &SkillTarget{
				Strategy: ManagedBlock,
				Path:     codexSkillPath,
				Render:   renderManagedInner,
			},
		},
	}
}

// jsonMCP is a small constructor for the common JSON MCP target shape.
func jsonMCP(configPath func(env Env) (string, error), serversKey string) *MCPTarget {
	return &MCPTarget{
		ConfigPath: configPath,
		Format:     codec.JSON,
		ServersKey: []string{serversKey},
		EntryStyle: CommandArgsEnv,
	}
}

// detectByMarker builds a Detect func that reports whether the resolved marker
// path exists on disk.
func detectByMarker(resolve func(env Env) (string, error)) func(env Env) bool {
	return func(env Env) bool {
		path, err := resolve(env)
		if err != nil || path == "" {
			return false
		}

		_, statErr := os.Stat(path)

		return statErr == nil
	}
}

// --- detection markers (config dirs/files that imply the agent is installed) ---

func claudeCodeMarker(env Env) (string, error) { return filepath.Join(env.Home, ".claude"), nil }
func cursorMarker(env Env) (string, error)     { return filepath.Join(env.Home, ".cursor"), nil }
func geminiMarker(env Env) (string, error)     { return filepath.Join(env.Home, ".gemini"), nil }
func zedMarker(env Env) (string, error)        { return filepath.Join(env.Home, ".config", "zed"), nil }

func windsurfMarker(env Env) (string, error) {
	return filepath.Join(env.Home, ".codeium", "windsurf"), nil
}

func opencodeMarker(env Env) (string, error) {
	return filepath.Join(env.Home, ".config", "opencode"), nil
}

func claudeDesktopMarker(env Env) (string, error) {
	return filepath.Join(appDataDir(env), "Claude"), nil
}

func vscodeMarker(env Env) (string, error) {
	return vscodeUserDir(env), nil
}

func clineMarker(env Env) (string, error) {
	return filepath.Join(vscodeUserDir(env), "globalStorage", "saoudrizwan.claude-dev"), nil
}

func rooMarker(env Env) (string, error) {
	return filepath.Join(vscodeUserDir(env), "globalStorage", "rooveterinaryinc.roo-cline"), nil
}
