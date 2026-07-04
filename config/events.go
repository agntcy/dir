// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

// Events configures the in-process event bus.
type Events struct {
	SubscriberBufferSize int  `json:"subscriber_buffer_size,omitempty" mapstructure:"subscriber_buffer_size"`
	LogSlowConsumers     bool `json:"log_slow_consumers,omitempty"     mapstructure:"log_slow_consumers"`
	LogPublishedEvents   bool `json:"log_published_events,omitempty"   mapstructure:"log_published_events"`
}

// DefaultEvents returns an Events config populated with production-safe defaults.
func DefaultEvents() Events {
	return Events{
		SubscriberBufferSize: DefaultEventsSubscriberBufferSize,
		LogSlowConsumers:     DefaultEventsLogSlowConsumers,
		LogPublishedEvents:   DefaultEventsLogPublishedEvents,
	}
}
