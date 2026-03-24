// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"
)

func TestConfig_Validate_DisabledNoop(t *testing.T) {
	t.Parallel()

	c := Config{Enabled: false}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestConfig_Validate_EnabledRequiresTimeoutAndPath(t *testing.T) {
	t.Parallel()

	c := Config{Enabled: true}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error when timeout/path missing")
	}

	c = Config{
		Enabled: true,
		Timeout: time.Minute,
		CLIPath: "/bin/mcp-scanner",
	}

	if err := c.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
