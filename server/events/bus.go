// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"sync"

	eventsv1 "github.com/agntcy/dir/api/events/v1"
	"github.com/agntcy/dir/server/events/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/google/uuid"
)

var logger = logging.Logger("events")

// Subscription represents an active event listener.
type Subscription struct {
	id      string
	ch      chan *Event
	filters []Filter
	cancel  chan struct{}
}

// EventBus manages event distribution to subscribers.
// It provides a thread-safe pub/sub mechanism with filtering support.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]*Subscription
	config      config.Config
	metrics     Metrics
}

// NewEventBus creates a new event bus with default configuration.
func NewEventBus() *EventBus {
	return NewEventBusWithConfig(config.DefaultConfig())
}

// NewEventBusWithConfig creates a new event bus with custom configuration.
func NewEventBusWithConfig(cfg config.Config) *EventBus {
	return &EventBus{
		subscribers: make(map[string]*Subscription),
		config:      cfg,
	}
}

// Publish broadcasts an event to all matching subscribers.
// Events are validated before publishing and delivered only to subscribers
// whose filters match the event.
//
// If a subscriber's channel is full (slow consumer), the event is dropped
// for that subscriber and a warning is logged (if configured).
func (b *EventBus) Publish(event *Event) {
	// Validate event before publishing
	if err := event.Validate(); err != nil {
		logger.Error("Invalid event rejected", "error", err)

		return
	}

	b.metrics.PublishedTotal.Add(1)

	if b.config.LogPublishedEvents {
		logger.Debug("Event published",
			"event_id", event.ID,
			"type", event.Type,
			"resource_id", event.ResourceID)
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var delivered uint64

	var dropped uint64

	// Deliver to all matching subscribers
	for _, sub := range b.subscribers {
		if Matches(event, sub.filters) {
			select {
			case sub.ch <- event:
				delivered++
			case <-sub.cancel:
				// Subscription was cancelled, skip
			default:
				// Channel is full (slow consumer)
				dropped++

				if b.config.LogSlowConsumers {
					logger.Warn("Dropped event due to slow consumer",
						"subscription_id", sub.id,
						"event_type", event.Type,
						"event_id", event.ID)
				}
			}
		}
	}

	b.metrics.DeliveredTotal.Add(delivered)

	if dropped > 0 {
		b.metrics.DroppedTotal.Add(dropped)
	}
}

// Subscribe creates a new subscription with the specified filters.
// Returns a unique subscription ID and a channel for receiving events.
//
// The caller is responsible for calling Unsubscribe when done to clean up resources.
//
// Example:
//
//	req := &eventsv1.ListenRequest{
//	    EventTypes: []eventsv1.EventType{eventsv1.EVENT_TYPE_RECORD_PUSHED},
//	}
//	subID, eventCh := bus.Subscribe(req)
//	defer bus.Unsubscribe(subID)
//
//	for event := range eventCh {
//	    // Process event
//	}
func (b *EventBus) Subscribe(req *eventsv1.ListenRequest) (string, <-chan *Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := uuid.New().String()
	sub := &Subscription{
		id:      id,
		ch:      make(chan *Event, b.config.SubscriberBufferSize),
		filters: BuildFilters(req),
		cancel:  make(chan struct{}),
	}

	b.subscribers[id] = sub
	b.metrics.SubscribersTotal.Add(1)

	logger.Info("New subscription created",
		"subscription_id", id,
		"event_types", req.GetEventTypes(),
		"label_filters", req.GetLabelFilters(),
		"cid_filters", req.GetCidFilters())

	return id, sub.ch
}

// Unsubscribe removes a subscription and cleans up resources.
// The event channel will be closed.
//
// It is safe to call Unsubscribe multiple times with the same ID or
// with an ID that doesn't exist.
func (b *EventBus) Unsubscribe(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sub, ok := b.subscribers[id]
	if !ok {
		return
	}

	// Signal cancellation and close channel
	close(sub.cancel)
	close(sub.ch)

	delete(b.subscribers, id)
	b.metrics.SubscribersTotal.Add(-1)

	logger.Info("Subscription removed", "subscription_id", id)
}

// SubscriberCount returns the current number of active subscribers.
func (b *EventBus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.subscribers)
}

// GetMetrics returns a snapshot of current metrics.
// This creates a copy with the current values.
func (b *EventBus) GetMetrics() MetricsSnapshot {
	return MetricsSnapshot{
		PublishedTotal:   b.metrics.PublishedTotal.Load(),
		DeliveredTotal:   b.metrics.DeliveredTotal.Load(),
		DroppedTotal:     b.metrics.DroppedTotal.Load(),
		SubscribersTotal: b.metrics.SubscribersTotal.Load(),
	}
}
