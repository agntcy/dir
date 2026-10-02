// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package pruneuntrusted

import (
	"strings"
	"time"
)

const (
	// DefaultInterval matches the prune-untrusted CronJob schedule (every 30 minutes).
	DefaultInterval = 30 * time.Minute

	// DefaultRecordTimeout is the per-record store delete timeout.
	DefaultRecordTimeout = 30 * time.Second

	// DefaultScanSeverity is the scan-severity threshold from the CronJob
	// (MEDIUM or worse).
	DefaultScanSeverity = "MEDIUM"
)

// Config holds configuration for the prune-untrusted reconciler task.
// The task deletes records that are not signature-trusted and whose highest
// scan severity meets or exceeds ScanSeverity.
type Config struct {
	// Enabled determines if the prune-untrusted task should run.
	// Defaults to false: this task deletes records.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to run the prune.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// RecordTimeout is the timeout for each individual store delete.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`

	// ScanSeverity is the minimum scan severity that, combined with
	// trusted:false, selects a record for deletion (e.g. "MEDIUM").
	ScanSeverity string `json:"scan_severity,omitempty" mapstructure:"scan_severity"`
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

// GetScanSeverity returns the scan-severity threshold with default fallback.
func (c *Config) GetScanSeverity() string {
	if c.ScanSeverity == "" {
		return DefaultScanSeverity
	}

	return strings.ToUpper(c.ScanSeverity)
}
