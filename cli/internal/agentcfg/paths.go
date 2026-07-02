// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ResolveEnv captures the ambient environment for descriptor resolvers.
func ResolveEnv() Env {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	return Env{Home: home, GOOS: runtime.GOOS, Cwd: cwd}
}

// vscodeUserDir returns the VS Code "User" config directory for the platform.
// This is the parent of mcp.json, prompts/, and globalStorage/.
func vscodeUserDir(env Env) string {
	switch env.GOOS {
	case "darwin":
		return filepath.Join(env.Home, "Library", "Application Support", "Code", "User")
	case "windows":
		return filepath.Join(env.Home, "AppData", "Roaming", "Code", "User")
	default:
		return filepath.Join(env.Home, ".config", "Code", "User")
	}
}

// appDataDir returns the per-platform directory GUI apps store config under.
func appDataDir(env Env) string {
	switch env.GOOS {
	case "darwin":
		return filepath.Join(env.Home, "Library", "Application Support")
	case "windows":
		return filepath.Join(env.Home, "AppData", "Roaming")
	default:
		return filepath.Join(env.Home, ".config")
	}
}

func requireHome(env Env) error {
	if env.Home == "" {
		return fmt.Errorf("could not determine home directory")
	}

	return nil
}

// --- MCP config paths ---

func claudeCodeMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".claude.json"), nil
}

func claudeDesktopMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(appDataDir(env), "Claude", "claude_desktop_config.json"), nil
}

func cursorMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".cursor", "mcp.json"), nil
}

func vscodeMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(vscodeUserDir(env), "mcp.json"), nil
}

func windsurfMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".codeium", "windsurf", "mcp_config.json"), nil
}

func clineMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(vscodeUserDir(env), "globalStorage",
		"saoudrizwan.claude-dev", "settings", "cline_mcp_settings.json"), nil
}

func rooMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(vscodeUserDir(env), "globalStorage",
		"rooveterinaryinc.roo-cline", "settings", "mcp_settings.json"), nil
}

func geminiMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".gemini", "settings.json"), nil
}

func opencodeMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".config", "opencode", "opencode.json"), nil
}

func zedMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".config", "zed", "settings.json"), nil
}

func continueMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".continue", "config.yaml"), nil
}

func codexMCPPath(env Env) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".codex", "config.toml"), nil
}

// --- skill / rules paths ---

func claudeCodeSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".claude", "skills", slug, "SKILL.md"), nil
}

func zedSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".agents", "skills", slug, "SKILL.md"), nil
}

func copilotSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(vscodeUserDir(env), "prompts", slug+".instructions.md"), nil
}

func windsurfSkillPath(env Env, _ string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".codeium", "windsurf", "memories", "global_rules.md"), nil
}

func clineSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	// Cline reads global rules from ~/Documents/Cline/Rules on macOS/Windows and
	// ~/Cline/Rules on Linux.
	if env.GOOS == "linux" {
		return filepath.Join(env.Home, "Cline", "Rules", slug+".md"), nil
	}

	return filepath.Join(env.Home, "Documents", "Cline", "Rules", slug+".md"), nil
}

func rooSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".roo", "rules", slug+".md"), nil
}

func continueSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".continue", "rules", slug+".md"), nil
}

func geminiSkillPath(env Env, _ string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".gemini", "GEMINI.md"), nil
}

func codexSkillPath(env Env, _ string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".codex", "AGENTS.md"), nil
}

func opencodeSkillPath(env Env, _ string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".config", "opencode", "AGENTS.md"), nil
}

// cursorNoGlobalSkill signals that Cursor has no global rules mechanism, so the
// engine falls back to the project path.
func cursorNoGlobalSkill(_ Env, _ string) (string, error) {
	return "", ErrNoGlobalPath
}

func cursorProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Cursor project rule")
	}

	return filepath.Join(env.Cwd, ".cursor", "rules", slug+".mdc"), nil
}

func codexMarker(env Env) (string, error)    { return filepath.Join(env.Home, ".codex"), nil }
func continueMarker(env Env) (string, error) { return filepath.Join(env.Home, ".continue"), nil }
