// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	"github.com/agntcy/dir/config/naming"
)

const (
	DefaultNameInterval      = 1 * time.Hour
	DefaultNameRecordTimeout = 30 * time.Second
)

// Name configures the DNS/well-known name verification reconciliation task.
type Name struct {
	Enabled       bool          `json:"enabled,omitempty"        mapstructure:"enabled"`
	Interval      time.Duration `json:"interval,omitempty"       mapstructure:"interval"`
	TTL           time.Duration `json:"ttl,omitempty"            mapstructure:"ttl"`
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`
}

func (c *Name) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultNameInterval
	}

	return c.Interval
}

// GetTTL returns the TTL, defaulting to the naming package TTL (shared with the API server).
func (c *Name) GetTTL() time.Duration {
	if c.TTL == 0 {
		return naming.DefaultTTL
	}

	return c.TTL
}

func (c *Name) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultNameRecordTimeout
	}

	return c.RecordTimeout
}
