// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package naming

import "time"

const (
	DefaultTTL     = 7 * 24 * time.Hour
	DefaultEnabled = true
)

// Naming configures the name verification cache used by the apiserver naming API.
// The reconciler "name" task performs re-verification using the same TTL.
type Naming struct {
	Enabled bool          `json:"enabled,omitempty" mapstructure:"enabled"`
	TTL     time.Duration `json:"ttl,omitempty"     mapstructure:"ttl"`
}

// GetTTL returns the TTL with default fallback.
func (c *Naming) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultTTL
	}

	return c.TTL
}
