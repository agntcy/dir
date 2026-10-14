// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"strings"
	"time"
)

const (
	// DefaultInterval is how often a run republishes matching local records.
	DefaultInterval = 24 * time.Hour

	// DefaultLimit is the search page size. A run pages through --offset
	// until a short page, same as the dirctl publish CronJob.
	DefaultLimit = 100
)

// Criteria is the local-store search filter the publish applies.
type Criteria struct {
	// Domains selects records that have any of these routing domains.
	// Empty means every indexed record.
	Domains []string `json:"domains,omitempty" mapstructure:"domains"`
}

// Config holds configuration for the publish reconciler task.
// The task searches the local index for records matching Criteria and
// submits one publication request per page, the same pipeline as the
// dirctl publish CronJob: search, then routing publish.
type Config struct {
	// Enabled determines if the publish task should run.
	// Defaults to false: this task announces records to the network.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run the search-and-publish.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// Criteria selects which local records to publish.
	Criteria Criteria `json:"criteria,omitzero" mapstructure:"criteria"`

	// Limit is the search page size. A run pages until a short page.
	Limit int `json:"limit,omitempty" mapstructure:"limit"`

	// DryRun logs the matching CIDs and skips CreatePublication.
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

// GetLimit returns the page size with default fallback.
func (c *Config) GetLimit() int {
	if c.Limit <= 0 {
		return DefaultLimit
	}

	return c.Limit
}

// GetDomains returns the configured domains, trimmed and without empties.
// An empty list means no domain filter.
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

	return out
}

// Validate is reserved for criteria that can be wrong. Empty domains are
// valid: the task then publishes every indexed record.
func (c *Config) Validate() error {
	return nil
}
