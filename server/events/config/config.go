// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	// DefaultSubscriberBufferSize is the default channel buffer size per subscriber.
	DefaultSubscriberBufferSize = 100

	// DefaultLogSlowConsumers is the default setting for logging slow consumers.
	DefaultLogSlowConsumers = true

	// DefaultLogPublishedEvents is the default setting for logging all published events.
	DefaultLogPublishedEvents = false
)

// Config holds event system configuration.
type Config struct {
	// SubscriberBufferSize is the channel buffer size per subscriber.
	// Larger buffers allow subscribers to fall behind temporarily without
	// dropping events, but use more memory.
	// Default: 100
	SubscriberBufferSize int

	// LogSlowConsumers enables logging when events are dropped due to
	// full subscriber buffers (slow consumers).
	// Default: true
	LogSlowConsumers bool

	// LogPublishedEvents enables debug logging of all published events.
	// This can be very verbose in production.
	// Default: false
	LogPublishedEvents bool
}

// DefaultConfig returns the default event system configuration.
func DefaultConfig() Config {
	return Config{
		SubscriberBufferSize: DefaultSubscriberBufferSize,
		LogSlowConsumers:     DefaultLogSlowConsumers,
		LogPublishedEvents:   DefaultLogPublishedEvents,
	}
}
