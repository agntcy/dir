// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prune

import (
	"fmt"
	"strings"
	"time"

	securityv1 "github.com/agntcy/dir/api/security/v1"
)

const (
	// DefaultInterval matches the prune-untrusted CronJob schedule (every 30 minutes).
	DefaultInterval = 30 * time.Minute

	// DefaultRecordTimeout is the per-record store delete timeout.
	DefaultRecordTimeout = 30 * time.Second

	// DefaultMinSeverity is the scan-severity threshold from the CronJob
	// (MEDIUM or worse).
	DefaultMinSeverity = securityv1.ScanSeverityMedium

	// DefaultLimit is how many matching records a run deletes. Same as the
	// CronJob: a run does not paginate, it converges over later intervals.
	DefaultLimit = 100

	// DefaultOlderThan is how old a record must be before prune will delete it.
	DefaultOlderThan = 168 * time.Hour
)

// Criteria is the search filter the prune applies.
type Criteria struct {
	// Trusted selects records by signature-verification status.
	// Config loading defaults this to false (not trusted).
	Trusted bool `json:"trusted" mapstructure:"trusted"`

	// MinSeverity is the minimum scan severity that selects a record
	// for deletion (e.g. "MEDIUM").
	MinSeverity string `json:"min_severity,omitempty" mapstructure:"min_severity"`

	// OlderThan is how long a record must have been in this index before
	// it is eligible to prune. Measured from records.created_at (first seen),
	// not the record's self-declared OASF created_at.
	OlderThan time.Duration `json:"older_than,omitempty" mapstructure:"older_than"`
}

// Config holds configuration for the prune reconciler task.
// The task deletes records that match Criteria.
type Config struct {
	// Enabled determines if the prune task should run.
	// Defaults to false: this task deletes records.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run the prune.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// RecordTimeout is the timeout for each individual store delete.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`

	// Criteria is the search filter: trusted status, min scan severity, and age.
	Criteria Criteria `json:"criteria,omitzero" mapstructure:"criteria"`

	// Limit is the maximum number of matching records deleted per run.
	Limit int `json:"limit,omitempty" mapstructure:"limit"`

	// DryRun logs the matching CIDs and skips store and index deletes.
	// Config loading defaults this to true.
	DryRun bool `json:"dry_run,omitempty" mapstructure:"dry_run"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultInterval
	}

	return c.Interval
}

// GetRecordTimeout returns the per-record timeout with default fallback.
func (c *Config) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultRecordTimeout
	}

	return c.RecordTimeout
}

// GetMinSeverity returns the scan-severity threshold with default fallback.
func (c *Config) GetMinSeverity() string {
	if c.Criteria.MinSeverity == "" {
		return DefaultMinSeverity
	}

	return strings.ToUpper(c.Criteria.MinSeverity)
}

// GetOlderThan returns the minimum record age with default fallback.
func (c *Config) GetOlderThan() time.Duration {
	if c.Criteria.OlderThan <= 0 {
		return DefaultOlderThan
	}

	return c.Criteria.OlderThan
}

// GetLimit returns the per-run delete cap with default fallback.
func (c *Config) GetLimit() int {
	if c.Limit <= 0 {
		return DefaultLimit
	}

	return c.Limit
}

// Validate reports an invalid min_severity. Unknown values must not reach
// the search layer: ScanSeveritiesGTE returns nil for them and the SQL
// predicate is dropped, so prune would match every record that satisfies
// the other criteria.
func (c *Config) Validate() error {
	severity := c.GetMinSeverity()
	if !securityv1.IsScanSeverity(severity) {
		return fmt.Errorf("invalid min_severity %q, must be one of: %s",
			severity, strings.Join(securityv1.ScanSeveritiesGTE(securityv1.ScanSeverityNone), ", "))
	}

	return nil
}
