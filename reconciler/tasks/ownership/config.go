// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ownership

import "time"

const (
	DefaultInterval = 5 * time.Minute
)

// Config holds configuration for the ownership reconciler task.
type Config struct {
	Enabled  bool          `json:"enabled,omitempty"  mapstructure:"enabled"`
	Interval time.Duration `json:"interval,omitempty" mapstructure:"interval"`

	// TrustedCACertFile is an optional path to a PEM-encoded CA certificate used
	// to verify the X.509 certificate embedded in signed ownership claims.
	// When set, signed claims whose certificate does not chain to this CA are rejected.
	// When unset, certificate chain verification is skipped (dev/insecure mode).
	TrustedCACertFile string `json:"trusted_ca_cert_file,omitempty" mapstructure:"trusted_ca_cert_file"`
}

func (c Config) GetInterval() time.Duration {
	if c.Interval <= 0 {
		return DefaultInterval
	}

	return c.Interval
}
