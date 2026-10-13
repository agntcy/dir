// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"errors"
	"strings"
	"time"
)

const (
	// DefaultInterval matches the nightly sync-netsec CronJob (0 3 * * *).
	DefaultInterval = 24 * time.Hour

	// DefaultLimit is how many routing hits a run considers. Same as the
	// CronJob: a run does not paginate, it converges over later intervals.
	DefaultLimit = 100

	// DefaultDomain is the routing domain used when none are configured.
	DefaultDomain = "network_security"
)

// Criteria is the routing search filter the sync applies.
type Criteria struct {
	// Domains is the routing domain queries (e.g. "network_security").
	// A record matching any of them is included. A YAML scalar or a
	// comma-separated env value is accepted as a one-element list.
	Domains []string `json:"domains,omitempty" mapstructure:"domains"`
}

// Config holds configuration for the sync reconciler task.
// The task searches the routing network for records matching Criteria and
// creates one sync per announcing peer.
type Config struct {
	// Enabled determines if the sync task should run.
	// Defaults to false: this task creates syncs against remote peers.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run the search-and-sync.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// Criteria selects which remote records to sync.
	Criteria Criteria `json:"criteria,omitzero" mapstructure:"criteria"`

	// Limit is the maximum number of routing search hits considered per run.
	Limit int `json:"limit,omitempty" mapstructure:"limit"`

	// DryRun logs the peers and CIDs and skips CreateSync.
	// Config loading defaults this to true.
	DryRun bool `json:"dry_run,omitempty" mapstructure:"dry_run"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return DefaultInterval
	}

	return c.Interval
}

// GetDomains returns the routing domains, trimmed and without empties.
// An empty list falls back to DefaultDomain.
func (c *Config) GetDomains() []string {
	seen := make(map[string]struct{}, len(c.Criteria.Domains))
	out := make([]string, 0, len(c.Criteria.Domains))

	for _, d := range c.Criteria.Domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}

		if _, ok := seen[d]; ok {
			continue
		}

		seen[d] = struct{}{}
		out = append(out, d)
	}

	if len(out) == 0 {
		return []string{DefaultDomain}
	}

	return out
}

// GetLimit returns the per-run search cap with default fallback.
func (c *Config) GetLimit() int {
	if c.Limit <= 0 {
		return DefaultLimit
	}

	return c.Limit
}

// Validate reports an empty domain list after defaults are applied.
func (c *Config) Validate() error {
	if len(c.GetDomains()) == 0 {
		return errors.New("at least one domain is required")
	}

	return nil
}
