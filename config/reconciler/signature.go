// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package reconciler

import "time"

const (
	DefaultSignatureInterval      = 1 * time.Minute
	DefaultSignatureTTL           = 7 * 24 * time.Hour
	DefaultSignatureRecordTimeout = 30 * time.Second
)

// Signature configures the signature verification reconciliation task.
type Signature struct {
	Enabled       bool          `json:"enabled,omitempty"        mapstructure:"enabled"`
	Interval      time.Duration `json:"interval,omitempty"       mapstructure:"interval"`
	TTL           time.Duration `json:"ttl,omitempty"            mapstructure:"ttl"`
	RecordTimeout time.Duration `json:"record_timeout,omitempty" mapstructure:"record_timeout"`
}

func (c *Signature) GetInterval() time.Duration {
	if c.Interval == 0 {
		return DefaultSignatureInterval
	}

	return c.Interval
}

func (c *Signature) GetTTL() time.Duration {
	if c.TTL == 0 {
		return DefaultSignatureTTL
	}

	return c.TTL
}

func (c *Signature) GetRecordTimeout() time.Duration {
	if c.RecordTimeout == 0 {
		return DefaultSignatureRecordTimeout
	}

	return c.RecordTimeout
}
