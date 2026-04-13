// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	enricherconfig "github.com/agntcy/dir/importer/enricher/config"
	scannerconfig "github.com/agntcy/dir/importer/scanner/config"
)

func TestConfig_Validate_MissingType(t *testing.T) {
	t.Parallel()

	c := Config{RegistryURL: "https://x.com"}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty Type")
	}
}

func TestConfig_Validate_MissingURL(t *testing.T) {
	t.Parallel()

	c := Config{Type: ImportTypeMCPRegistry}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty RegistryURL")
	}
}

func TestConfig_Validate_FileMissingPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "mcphost.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		Type: ImportTypeMCP,
		Enricher: enricherconfig.Config{
			ConfigFile:        cfgPath,
			RequestsPerMinute: 1,
		},
		Scanner: scannerconfig.Config{Enabled: false},
	}

	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty FilePath")
	}
}

func TestConfig_Validate_FileOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "mcphost.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		Type:     ImportTypeMCP,
		FilePath: filepath.Join(dir, "server.json"),
		Enricher: enricherconfig.Config{
			ConfigFile:        cfgPath,
			RequestsPerMinute: 1,
		},
		Scanner: scannerconfig.Config{Enabled: false},
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_A2AFileOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "mcphost.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		Type:     ImportTypeA2A,
		FilePath: filepath.Join(dir, "agent.json"),
		Enricher: enricherconfig.Config{
			ConfigFile:        cfgPath,
			RequestsPerMinute: 1,
		},
		Scanner: scannerconfig.Config{Enabled: false},
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_AgentSkillFileOK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "mcphost.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		Type:     ImportTypeAgentSkill,
		FilePath: filepath.Join(dir, "my-skill"),
		Enricher: enricherconfig.Config{
			ConfigFile:        cfgPath,
			RequestsPerMinute: 1,
		},
		Scanner: scannerconfig.Config{Enabled: false},
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_OK(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "mcphost.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		Type:        ImportTypeMCPRegistry,
		RegistryURL: "https://registry.example.com",
		Enricher: enricherconfig.Config{
			ConfigFile:        cfgPath,
			RequestsPerMinute: 1,
		},
		Scanner: scannerconfig.Config{Enabled: false},
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
