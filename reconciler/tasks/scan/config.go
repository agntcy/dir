// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package scan

import (
	"time"

	"github.com/agntcy/dir/server/types"
)

const (
	// DefaultInterval is the default scan reconciliation interval.
	// Scanning is LLM-backed and therefore expensive; 6 hours balances freshness with cost.
	DefaultInterval = 6 * time.Hour

	// DefaultTTL is how long a scan result is considered fresh before re-scanning.
	DefaultTTL = 7 * 24 * time.Hour

	// DefaultRecordTimeout is the per-record timeout for the full scan (clone + scanner binary).
	DefaultRecordTimeout = 5 * time.Minute

	// DefaultMCPCLIPath is the default binary name for the MCP scanner.
	DefaultMCPCLIPath = "mcp-scanner"

	// DefaultSkillCLIPath is the default binary name for the skill scanner.
	DefaultSkillCLIPath = "skill-scanner"

	// DefaultA2ACLIPath is the default binary name for the A2A scanner.
	DefaultA2ACLIPath = "a2a-scanner"
)

// Config holds configuration for the security scan reconciliation task.
type Config struct {
	// Enabled determines if the scan task should run. Defaults to false (opt-in).
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run the scan task.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// TTL is the time-to-live for scan results; records are re-scanned after expiry.
	TTL time.Duration `json:"ttl,omitempty" mapstructure:"ttl"`

	// RecordTimeout is the timeout for each individual record scan (all runners combined).
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`

	// MCPCLIPath is the path to the mcp-scanner binary.
	MCPCLIPath string `json:"mcp_cli_path,omitempty" mapstructure:"mcp_cli_path"`

	// SkillCLIPath is the path to the skill-scanner binary.
	SkillCLIPath string `json:"skill_cli_path,omitempty" mapstructure:"skill_cli_path"`

	// A2ACLIPath is the path to the a2a-scanner binary.
	A2ACLIPath string `json:"a2a_cli_path,omitempty" mapstructure:"a2a_cli_path"`

	// DisableEndpointScan turns off the MCP live-endpoint scan phase, leaving
	// only the source-code scan. Set it where the reconciler must not make
	// outbound connections to the endpoints named in published records.
	DisableEndpointScan bool `json:"disable_endpoint_scan,omitempty" mapstructure:"disable_endpoint_scan"`

	// AllowPrivateEndpoints permits MCP endpoints on loopback, link-local,
	// private, and other reserved ranges. Off by default so the safe posture
	// ships out of the box; self-hosted deployments that run the directory and
	// its MCP servers on one private network need it on.
	AllowPrivateEndpoints bool `json:"allow_private_endpoints,omitempty" mapstructure:"allow_private_endpoints"`

	// AllowInsecureTransport permits plain http MCP endpoints. Separate from
	// AllowPrivateEndpoints so that reaching a private-range endpoint does not
	// also opt into cleartext scans of arbitrary public hosts.
	AllowInsecureTransport bool `json:"allow_insecure_transport,omitempty" mapstructure:"allow_insecure_transport"`

	// MaxEndpointsPerRecord bounds how many distinct MCP endpoints a single
	// record can make the reconciler dial. Zero applies the default. Raise it
	// for aggregator records that legitimately bundle several MCP servers;
	// note that each endpoint costs one scanner invocation per subcommand.
	MaxEndpointsPerRecord int `json:"max_endpoints_per_record,omitempty" mapstructure:"max_endpoints_per_record"`
}

// ScanSchedule returns the retry schedule applied to scan result rows.
//
// The backoff base is the scan interval, so a first failure retries on the
// next pass and only repeated ones back off. The schedule is materialised into
// each row as it is written, so changing the interval does not reschedule
// existing rows; they adopt it on their next attempt.
func (c *Config) ScanSchedule() types.ScanSchedule {
	return types.ScanSchedule{
		FreshFor:  c.GetTTL(),
		RetryBase: c.GetInterval(),
		RetryMax:  types.DefaultScanRetryMax,
	}
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}

// GetTTL returns the TTL with default fallback.
func (c *Config) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}

	return c.TTL
}

// GetRecordTimeout returns the per-record timeout with default fallback.
func (c *Config) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultRecordTimeout
	}

	return c.RecordTimeout
}

// GetMCPCLIPath returns the MCP scanner binary path with default fallback.
func (c *Config) GetMCPCLIPath() string {
	if c.MCPCLIPath == "" {
		return DefaultMCPCLIPath
	}

	return c.MCPCLIPath
}

// GetSkillCLIPath returns the skill scanner binary path with default fallback.
func (c *Config) GetSkillCLIPath() string {
	if c.SkillCLIPath == "" {
		return DefaultSkillCLIPath
	}

	return c.SkillCLIPath
}

// GetA2ACLIPath returns the A2A scanner binary path with default fallback.
func (c *Config) GetA2ACLIPath() string {
	if c.A2ACLIPath == "" {
		return DefaultA2ACLIPath
	}

	return c.A2ACLIPath
}
