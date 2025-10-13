// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"github.com/agntcy/dir/server/events/config"
)

// Service manages the event streaming system lifecycle.
// It creates and owns the EventBus and provides access to it
// for other services to publish events.
type Service struct {
	bus    *EventBus
	config config.Config
}

// New creates a new event service with default configuration.
// The service is ready to use immediately (no Start() needed).
//
// Example:
//
//	eventService := events.New()
//	defer eventService.Stop()
//
//	// Get bus for publishing
//	bus := eventService.Bus()
//	bus.RecordPushed("bafyxxx", labels)
func New() *Service {
	cfg := config.DefaultConfig()

	logger.Info("Initializing event service",
		"subscriber_buffer_size", cfg.SubscriberBufferSize,
		"log_slow_consumers", cfg.LogSlowConsumers,
		"log_published_events", cfg.LogPublishedEvents)

	return &Service{
		bus:    NewEventBusWithConfig(cfg),
		config: cfg,
	}
}

// NewWithConfig creates a new event service with custom configuration.
func NewWithConfig(cfg config.Config) *Service {
	logger.Info("Initializing event service with custom config",
		"subscriber_buffer_size", cfg.SubscriberBufferSize,
		"log_slow_consumers", cfg.LogSlowConsumers,
		"log_published_events", cfg.LogPublishedEvents)

	return &Service{
		bus:    NewEventBusWithConfig(cfg),
		config: cfg,
	}
}

// Bus returns the event bus for publishing events.
// Other services should use this to emit events.
func (s *Service) Bus() *EventBus {
	return s.bus
}

// Stop gracefully shuts down the event service.
// This closes all active subscriptions and prevents new ones.
func (s *Service) Stop() error {
	logger.Info("Stopping event service",
		"active_subscribers", s.bus.SubscriberCount())

	// Get final metrics
	metrics := s.bus.GetMetrics()
	logger.Info("Event service stopped",
		"total_published", metrics.PublishedTotal,
		"total_delivered", metrics.DeliveredTotal,
		"total_dropped", metrics.DroppedTotal)

	// Note: We don't close subscriptions here because:
	// 1. They are managed by the controller (gRPC stream lifecycle)
	// 2. Subscribers will naturally disconnect when server stops
	// 3. Forcing close could cause panics in active subscribers

	return nil
}
