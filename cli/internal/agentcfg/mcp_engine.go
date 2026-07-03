// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package agentcfg

import (
	"fmt"
	"os"
	"reflect"

	"github.com/agntcy/dir/cli/internal/agentcfg/codec"
	"github.com/agntcy/dir/cli/internal/agentcfg/fsutil"
)

const configFilePerm = 0o600

// InstallMCP upserts the MCP server entry under serverName into the agent's
// config file, preserving all sibling content. It is idempotent: re-running with
// an identical entry reports ActionUnchanged and writes nothing. When dryRun is
// set, it computes the action but does not write.
func InstallMCP(target *MCPTarget, env Env, entry map[string]any, serverName string, dryRun bool) (Outcome, error) {
	path, err := target.ConfigPath(env)
	if err != nil {
		return failOutcome(Outcome{Artifact: "mcp"}, fmt.Errorf("resolve mcp config path: %w", err))
	}

	outcome := Outcome{Artifact: "mcp", Path: path}

	m, err := loadConfig(target.Format, path)
	if err != nil {
		return failOutcome(outcome, err)
	}

	keyPath := append(append([]string{}, target.ServersKey...), serverName)

	existing, present := codec.GetNested(m, keyPath...)
	switch {
	case !present:
		outcome.Action = ActionAdded
	case reflect.DeepEqual(existing, entry):
		outcome.Action = ActionUnchanged
	default:
		outcome.Action = ActionUpdated
	}

	if dryRun || outcome.Action == ActionUnchanged {
		return outcome, nil
	}

	codec.SetNested(m, entry, keyPath...)

	if err := writeConfig(target.Format, path, m); err != nil {
		return failOutcome(outcome, err)
	}

	return outcome, nil
}

// RemoveMCP deletes only the serverName entry from the agent's config file,
// preserving all sibling servers. It never deletes the config file itself. An
// absent entry (or missing file) reports ActionUnchanged, so uninstall is
// idempotent. When dryRun is set, it computes the action but does not write.
func RemoveMCP(target *MCPTarget, env Env, serverName string, dryRun bool) (Outcome, error) {
	path, err := target.ConfigPath(env)
	if err != nil {
		return failOutcome(Outcome{Artifact: "mcp"}, fmt.Errorf("resolve mcp config path: %w", err))
	}

	outcome := Outcome{Artifact: "mcp", Path: path}

	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	m, err := loadConfig(target.Format, path)
	if err != nil {
		return failOutcome(outcome, err)
	}

	keyPath := append(append([]string{}, target.ServersKey...), serverName)

	if _, present := codec.GetNested(m, keyPath...); !present {
		outcome.Action = ActionUnchanged

		return outcome, nil
	}

	outcome.Action = ActionRemoved

	if dryRun {
		return outcome, nil
	}

	codec.DeleteNested(m, keyPath...)

	if err := writeConfig(target.Format, path, m); err != nil {
		return failOutcome(outcome, err)
	}

	return outcome, nil
}

// writeConfig encodes the map and writes it atomically.
func writeConfig(format codec.Format, path string, m map[string]any) error {
	data, err := codec.Encode(format, m)
	if err != nil {
		return fmt.Errorf("encode config %s: %w", path, err)
	}

	if err := fsutil.WriteAtomic(path, data, configFilePerm); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}

	return nil
}

// MCPEntryPresent reports whether our server entry already exists in the config.
func MCPEntryPresent(target *MCPTarget, env Env, serverName string) bool {
	path, err := target.ConfigPath(env)
	if err != nil {
		return false
	}

	m, err := loadConfig(target.Format, path)
	if err != nil {
		return false
	}

	keyPath := append(append([]string{}, target.ServersKey...), serverName)
	_, present := codec.GetNested(m, keyPath...)

	return present
}

// loadConfig reads and decodes a config file, treating a missing file as empty.
func loadConfig(format codec.Format, path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}

		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	m, err := codec.Decode(format, data)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}

	return m, nil
}
