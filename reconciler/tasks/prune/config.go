// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package prune

import (
	"errors"
	"fmt"
	"strings"
	"time"

	securityv1 "github.com/agntcy/dir/api/security/v1"
	"github.com/agntcy/dir/server/types"
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
	// Trusted selects records by signature status: true selects [verified],
	// false selects [failed, unsigned]. Unset means false. Mutually exclusive
	// with SignatureStatuses.
	Trusted *bool `json:"trusted,omitempty" mapstructure:"trusted"`

	// SignatureStatuses selects records whose signature status is any of
	// "verified", "failed" or "unsigned". Mutually exclusive with Trusted.
	//   - verified: at least one signature verifies.
	//   - failed: a signature verified before but no longer does (e.g. revoked).
	//   - unsigned: no signature is attached to the record.
	// A signed record with no verification result matches none of them, so
	// prune never deletes it, whether selecting by these or by Trusted.
	SignatureStatuses []string `json:"signature_statuses,omitempty" mapstructure:"signature_statuses"`

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

	// Criteria is the search filter: signature status (or trusted), min scan
	// severity, and age.
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

// GetTrusted returns the trusted criterion; unset means false.
func (c *Config) GetTrusted() bool {
	return c.Criteria.Trusted != nil && *c.Criteria.Trusted
}

// UsesSignatureStatuses reports whether SignatureStatuses is configured
// instead of Trusted.
func (c *Config) UsesSignatureStatuses() bool {
	return len(c.Criteria.SignatureStatuses) > 0
}

// GetSignatureStatuses returns the signature statuses prune selects. Trusted is
// an alias over them: true is [verified], false (or unset) is
// [failed, unsigned].
//
// This differs from the trusted search filter on purpose: "not trusted" there
// also lists signed records without a verification result, but prune never
// deletes those. A filter that deletes must be conservative about missing
// evidence; one that lists should not hide records.
func (c *Config) GetSignatureStatuses() []string {
	if c.UsesSignatureStatuses() {
		return c.Criteria.SignatureStatuses
	}

	if c.GetTrusted() {
		return []string{types.SignatureStatusVerified}
	}

	return []string{types.SignatureStatusFailed, types.SignatureStatusUnsigned}
}

// Validate reports an invalid min_severity or signature status, or both
// trusted and signature_statuses set. Unknown values
// must not reach the search layer: ScanSeveritiesGTE returns nil for them and
// the SQL predicate is dropped, so prune would match every record that
// satisfies the other criteria.
func (c *Config) Validate() error {
	severity := c.GetMinSeverity()
	if !securityv1.IsScanSeverity(severity) {
		return fmt.Errorf("invalid min_severity %q, must be one of: %s",
			severity, strings.Join(securityv1.ScanSeveritiesGTE(securityv1.ScanSeverityNone), ", "))
	}

	if c.Criteria.Trusted != nil && c.UsesSignatureStatuses() {
		return errors.New("set either trusted or signature_statuses, not both")
	}

	for _, s := range c.Criteria.SignatureStatuses {
		if !types.IsSignatureStatus(s) {
			return fmt.Errorf("invalid signature_statuses value %q, must be one of: %s",
				s, strings.Join(types.SignatureStatuses(), ", "))
		}
	}

	return nil
}
