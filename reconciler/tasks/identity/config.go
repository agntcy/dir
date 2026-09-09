// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package identity

import "time"

const (
	// DefaultInterval is the default reconciliation interval for identity/ownership claim verification.
	DefaultInterval = 5 * time.Minute
)

// Config holds the configuration for the identity/ownership claim
// reconciliation task.
type Config struct {
	// Enabled determines if the identity task should run.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Interval is how often to re-verify identity/ownership claims.
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// SpiffeTrustDomains maps a SPIFFE trust domain (e.g. "acme.com") to a path
	// containing one or more PEM-encoded CA certificates, used to validate the
	// certificate chain of SPIFFE identity/ownership claims for that domain.
	SpiffeTrustDomains map[string]string `json:"spiffe_trust_domains,omitempty" mapstructure:"spiffe_trust_domains"`
}

// GetInterval returns the interval with default fallback.
func (c *Config) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return DefaultInterval
	}

	return c.Interval
}
