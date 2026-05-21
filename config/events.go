// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

// Event bus defaults.
const (
	// DefaultEventsSubscriberBufferSize is the default channel buffer
	// size per subscriber.
	DefaultEventsSubscriberBufferSize = 100

	// DefaultEventsLogSlowConsumers logs subscribers that drop events.
	DefaultEventsLogSlowConsumers = true

	// DefaultEventsLogPublishedEvents logs every published event.
	// Very verbose; off by default.
	DefaultEventsLogPublishedEvents = false
)

// Events holds the configuration for the apiserver event bus.
type Events struct {
	// SubscriberBufferSize is the channel buffer size per subscriber.
	// Larger buffers tolerate temporary lag but use more memory.
	SubscriberBufferSize int `json:"subscriber_buffer_size,omitempty" mapstructure:"subscriber_buffer_size"`

	// LogSlowConsumers logs when events are dropped due to full
	// subscriber buffers.
	LogSlowConsumers bool `json:"log_slow_consumers,omitempty" mapstructure:"log_slow_consumers"`

	// LogPublishedEvents enables debug logging of all published events.
	LogPublishedEvents bool `json:"log_published_events,omitempty" mapstructure:"log_published_events"`
}

// DefaultEvents returns the default event bus configuration.
func DefaultEvents() Events {
	return Events{
		SubscriberBufferSize: DefaultEventsSubscriberBufferSize,
		LogSlowConsumers:     DefaultEventsLogSlowConsumers,
		LogPublishedEvents:   DefaultEventsLogPublishedEvents,
	}
}
