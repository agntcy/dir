// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import (
	"fmt"
	"time"

	ansresolver "github.com/agntcy/dir/client/utils/identity/resolvers/ans"
)

const (
	// DefaultInterval is the default reconciliation interval for claim verification.
	DefaultInterval = 1 * time.Hour

	// DefaultRecordTimeout is the default timeout for verifying all the claims of one record.
	DefaultRecordTimeout = 1 * time.Minute

	// ansStagesPerClaim is how many of the ans resolver's timeouts one ans://
	// claim may take: one for the publisher's DNS record, one for the log.
	ansStagesPerClaim = 2

	// ansClaimsPerRecord is how many ans:// claims one record can need to
	// verify within its budget: an identity claim and an ownership claim.
	ansClaimsPerRecord = 2

	// DefaultANSStaleGrace is how long an ans:// result outlives lookups that
	// get no answer. Shorter than the task's grace for the other schemes, since
	// the publisher's own DNS is among the lookups that may get none.
	DefaultANSStaleGrace = 24 * time.Hour
)

// TrustBundle is the X.509 trust bundle of one SPIFFE trust domain.
type TrustBundle struct {
	// TrustDomain is the trust domain, e.g. "acme.com".
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`

	// BundleFile is the path of the PEM file holding the trust domain's root certificates.
	BundleFile string `json:"bundle_file,omitempty" mapstructure:"bundle_file"`
}

// Config holds the configuration for the identity claim verification task.
type Config struct {
	// Enabled determines if the identity task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often every claim is verified again.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// RecordTimeout is the timeout for verifying all the claims of one record.
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`

	// SPIFFETrustBundles are the trust bundles a spiffe:// claim's certificate is
	// validated against. A trust domain with no bundle here fails closed. The files
	// are read on every run, so a rotated or revoked bundle applies on the next one.
	// YAML only: a list does not map onto environment variables.
	SPIFFETrustBundles []TrustBundle `json:"spiffe_trust_bundles,omitempty" mapstructure:"spiffe_trust_bundles"`

	// ANS configures verification of ans:// claims through the Agent Name
	// Service. Off unless enabled; an ans:// claim then fails as an unsupported
	// scheme.
	ANS ANSConfig `json:"ans" mapstructure:"ans"`
}

// ANSConfig is the ans:// block: the switch, the task's own grace for ans://
// results, and the resolver's settings, whose keys sit beside them under
// identity.ans.
type ANSConfig struct {
	// Enabled turns verification of ans:// subjects on.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// StaleGrace is how long a stored result outlives lookups that get no
	// answer: the publisher's DNS not answering, the trusted log unreachable.
	// It replaces the task's seven days for ans:// claims. Zero, which an
	// unset key leaves, means DefaultANSStaleGrace; a positive value below
	// Interval keeps nothing.
	StaleGrace time.Duration `json:"stale_grace,omitempty" mapstructure:"stale_grace"`

	ansresolver.Config `mapstructure:",squash"`
}

// GetStaleGrace returns the grace with default fallback: zero means the
// default, not no grace.
func (c *ANSConfig) GetStaleGrace() time.Duration {
	if c.StaleGrace == 0 {
		return DefaultANSStaleGrace
	}

	return c.StaleGrace
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

// validateANS checks the ans block against the task's own settings: a record
// may carry an ans:// identity claim and an ans:// ownership claim, each taking
// up to two of the resolver's timeouts, and all of that has to fit the record
// budget; the resolver's memo of a subject's attestation must expire between
// runs, or a revocation would wait for the run after next; and the grace must
// not be negative.
func (c *Config) validateANS() error {
	if !c.ANS.Enabled {
		return nil
	}

	if needed := ansClaimsPerRecord * ansStagesPerClaim * c.ANS.GetTimeout(); needed > c.GetRecordTimeout() {
		return fmt.Errorf("identity.ans.timeout %s needs identity.record_timeout of at least %s, got %s", c.ANS.GetTimeout(), needed, c.GetRecordTimeout())
	}

	if ttl := c.ANS.GetStatusCacheTTL(); ttl >= c.GetInterval() {
		return fmt.Errorf("identity.ans.status_cache_ttl %s must be below identity.interval %s", ttl, c.GetInterval())
	}

	if c.ANS.StaleGrace < 0 {
		return fmt.Errorf("identity.ans.stale_grace must not be negative, got %s", c.ANS.StaleGrace)
	}

	return nil
}

// trustDomains returns the bundle file of each trust domain.
func (c *Config) trustDomains() map[string]string {
	domains := make(map[string]string, len(c.SPIFFETrustBundles))
	for _, b := range c.SPIFFETrustBundles {
		domains[b.TrustDomain] = b.BundleFile
	}

	return domains
}
