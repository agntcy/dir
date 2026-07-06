// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"testing"
	"time"
)

func TestConfig_Defaults(t *testing.T) {
	t.Parallel()

	c := &Config{}

	if got := c.GetInterval(); got != DefaultInterval {
		t.Errorf("GetInterval() = %v, want %v", got, DefaultInterval)
	}

	if got := c.GetTTL(); got != DefaultTTL {
		t.Errorf("GetTTL() = %v, want %v", got, DefaultTTL)
	}

	if got := c.GetRecordTimeout(); got != DefaultRecordTimeout {
		t.Errorf("GetRecordTimeout() = %v, want %v", got, DefaultRecordTimeout)
	}

	if got := c.GetMCPCLIPath(); got != DefaultMCPCLIPath {
		t.Errorf("GetMCPCLIPath() = %q, want %q", got, DefaultMCPCLIPath)
	}

	if got := c.GetSkillCLIPath(); got != DefaultSkillCLIPath {
		t.Errorf("GetSkillCLIPath() = %q, want %q", got, DefaultSkillCLIPath)
	}
}

func TestConfig_CustomValues(t *testing.T) {
	t.Parallel()

	c := &Config{
		Interval:      2 * time.Hour,
		TTL:           24 * time.Hour,
		RecordTimeout: 10 * time.Minute,
		MCPCLIPath:    "/opt/mcp-scanner",
		SkillCLIPath:  "/opt/skill-scanner",
	}

	if got := c.GetInterval(); got != 2*time.Hour {
		t.Errorf("GetInterval() = %v, want 2h", got)
	}

	if got := c.GetTTL(); got != 24*time.Hour {
		t.Errorf("GetTTL() = %v, want 24h", got)
	}

	if got := c.GetRecordTimeout(); got != 10*time.Minute {
		t.Errorf("GetRecordTimeout() = %v, want 10m", got)
	}

	if got := c.GetMCPCLIPath(); got != "/opt/mcp-scanner" {
		t.Errorf("GetMCPCLIPath() = %q", got)
	}

	if got := c.GetSkillCLIPath(); got != "/opt/skill-scanner" {
		t.Errorf("GetSkillCLIPath() = %q", got)
	}
}
