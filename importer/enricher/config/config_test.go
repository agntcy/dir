// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate_MissingConfigFile(t *testing.T) {
	t.Parallel()

	c := Config{RequestsPerMinute: 1}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty ConfigFile")
	}
}

func TestConfig_Validate_InvalidRequestsPerMinute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "enricher.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		ConfigFile:        cfgPath,
		RequestsPerMinute: 0,
	}

	if err := c.Validate(); err == nil {
		t.Fatal("expected error for RequestsPerMinute <= 0")
	}
}

func TestConfig_Validate_EmbedsDefaultPrompts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	cfgPath := filepath.Join(dir, "enricher.json")
	if err := os.WriteFile(cfgPath, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	c := Config{
		ConfigFile:        cfgPath,
		RequestsPerMinute: 1,
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	if c.SkillsPromptTemplate == "" || c.DomainsPromptTemplate == "" {
		t.Fatal("expected default embedded prompts to be applied")
	}
}
