// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import "time"

const (
	// DefaultInterval is the default reconciliation interval for claim verification.
	DefaultInterval = 1 * time.Hour

	// DefaultRecordTimeout is the default timeout for verifying all the claims of one record.
	DefaultRecordTimeout = 1 * time.Minute
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

// trustDomains returns the bundle file of each trust domain.
func (c *Config) trustDomains() map[string]string {
	domains := make(map[string]string, len(c.SPIFFETrustBundles))
	for _, b := range c.SPIFFETrustBundles {
		domains[b.TrustDomain] = b.BundleFile
	}

	return domains
}
