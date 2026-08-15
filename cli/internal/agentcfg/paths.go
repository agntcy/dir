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

// --- project (repo/cwd) MCP config paths ---
// Only agents with a documented project-scope MCP location are listed here;
// agents without one leave ProjectConfigPath nil and are skipped-with-note.

func requireCwd(env Env, agent string) error {
	if env.Cwd == "" {
		return fmt.Errorf("cannot determine current directory for %s project config", agent)
	}

	return nil
}

func claudeCodeProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "Claude Code"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, ".mcp.json"), nil
}

func cursorProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "Cursor"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, ".cursor", "mcp.json"), nil
}

func vscodeProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "VS Code"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, ".vscode", "mcp.json"), nil
}

func rooProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "Roo Code"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, ".roo", "mcp.json"), nil
}

func geminiProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "Gemini CLI"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, ".gemini", "settings.json"), nil
}

func opencodeProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "OpenCode"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, "opencode.json"), nil
}

func zedProjectMCPPath(env Env) (string, error) {
	if err := requireCwd(env, "Zed"); err != nil {
		return "", err
	}

	return filepath.Join(env.Cwd, ".zed", "settings.json"), nil
}

// --- skill / rules paths ---

func claudeCodeSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".claude", "skills", slug, "SKILL.md"), nil
}

func claudeCodeProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Claude Code project skill")
	}

	return filepath.Join(env.Cwd, ".claude", "skills", slug, "SKILL.md"), nil
}

func continueProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Continue project skill")
	}

	return filepath.Join(env.Cwd, ".continue", "rules", slug+".md"), nil
}

func zedUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".agents", "skills", slug, "SKILL.md"), nil
}

func zedProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Zed project skill")
	}

	return filepath.Join(env.Cwd, ".agents", "skills", slug, "SKILL.md"), nil
}

func copilotUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".copilot", "skills", slug, "SKILL.md"), nil
}

func copilotProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Copilot project skill")
	}

	return filepath.Join(env.Cwd, ".github", "skills", slug, "SKILL.md"), nil
}

func windsurfUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".codeium", "windsurf", "skills", slug, "SKILL.md"), nil
}

func windsurfProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Windsurf project skill")
	}

	return filepath.Join(env.Cwd, ".windsurf", "skills", slug, "SKILL.md"), nil
}

func clineUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".cline", "skills", slug, "SKILL.md"), nil
}

func clineProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Cline project skill")
	}

	return filepath.Join(env.Cwd, ".cline", "skills", slug, "SKILL.md"), nil
}

func rooUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".roo", "skills", slug, "SKILL.md"), nil
}

func rooProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Roo Code project skill")
	}

	return filepath.Join(env.Cwd, ".roo", "skills", slug, "SKILL.md"), nil
}

func continueSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".continue", "rules", slug+".md"), nil
}

func geminiUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".gemini", "skills", slug, "SKILL.md"), nil
}

func geminiProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Gemini CLI project skill")
	}

	return filepath.Join(env.Cwd, ".gemini", "skills", slug, "SKILL.md"), nil
}

func codexUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".agents", "skills", slug, "SKILL.md"), nil
}

func codexProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Codex CLI project skill")
	}

	return filepath.Join(env.Cwd, ".agents", "skills", slug, "SKILL.md"), nil
}

func opencodeUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".config", "opencode", "skills", slug, "SKILL.md"), nil
}

func opencodeProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for OpenCode project skill")
	}

	return filepath.Join(env.Cwd, ".opencode", "skills", slug, "SKILL.md"), nil
}

func cursorUserSkillPath(env Env, slug string) (string, error) {
	if err := requireHome(env); err != nil {
		return "", err
	}

	return filepath.Join(env.Home, ".cursor", "skills", slug, "SKILL.md"), nil
}

func cursorProjectSkillPath(env Env, slug string) (string, error) {
	if env.Cwd == "" {
		return "", fmt.Errorf("cannot determine current directory for Cursor project skill")
	}

	return filepath.Join(env.Cwd, ".cursor", "skills", slug, "SKILL.md"), nil
}

func codexMarker(env Env) (string, error)    { return filepath.Join(env.Home, ".codex"), nil }
func continueMarker(env Env) (string, error) { return filepath.Join(env.Home, ".continue"), nil }
