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
	wg          sync.WaitGroup // Tracks in-flight publishAsync goroutines
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

// Publish broadcasts an event to all matching subscribers asynchronously.
// This method returns immediately without blocking the caller, making it safe
// to call from API handlers and other performance-critical code paths.
//
// Events are validated before publishing and delivered only to subscribers
// whose filters match the event. If a subscriber's channel is full (slow consumer),
// the event is dropped for that subscriber and a warning is logged (if configured).
//
// The actual delivery happens in a background goroutine, so there is no guarantee
// about the order or timing of delivery relative to other operations.
func (b *EventBus) Publish(event *Event) {
	// Validate event before publishing
	if err := event.Validate(); err != nil {
		logger.Error("Invalid event rejected", "error", err)

		return
	}

	b.metrics.PublishedTotal.Add(1)

	// Copy event to avoid race conditions when accessed from background goroutine.
	// The caller may reuse or modify the event struct after Publish returns.
	eventCopy := &Event{
		ID:         event.ID,
		Type:       event.Type,
		ResourceID: event.ResourceID,
		Labels:     append([]string(nil), event.Labels...),
		Metadata:   make(map[string]string, len(event.Metadata)),
		Timestamp:  event.Timestamp,
	}

	for k, v := range event.Metadata {
		eventCopy.Metadata[k] = v
	}

	// Track the async goroutine so Unsubscribe can wait for completion
	b.wg.Add(1)

	// Publish in background goroutine - returns immediately!
	// This ensures the caller (API handler) is never blocked by event delivery.
	go b.publishAsync(eventCopy)
}

// publishAsync handles the actual event delivery in a background goroutine.
// It takes a snapshot of subscribers while holding the lock briefly, then
// delivers events without holding any locks.
func (b *EventBus) publishAsync(event *Event) {
	defer b.wg.Done() // Signal completion when done

	if b.config.LogPublishedEvents {
		logger.Debug("Event published",
			"event_id", event.ID,
			"type", event.Type,
			"resource_id", event.ResourceID)
	}

	// Take a snapshot of subscribers while holding the lock briefly.
	// This minimizes lock contention - we only hold the lock long enough
	// to copy the subscriber list (~1µs), not during actual delivery.
	b.mu.RLock()
	snapshot := make([]*Subscription, 0, len(b.subscribers))

	for _, sub := range b.subscribers {
		snapshot = append(snapshot, sub)
	}

	b.mu.RUnlock()

	// Now deliver to all matching subscribers without holding any locks.
	// This prevents blocking Subscribe/Unsubscribe operations and allows
	// parallel event delivery.
	var delivered uint64

	var dropped uint64

	for _, sub := range snapshot {
		// Check if subscription was cancelled before attempting delivery.
		// This prevents most cases of sending to a closed channel.
		select {
		case <-sub.cancel:
			// Subscription was cancelled/closed, skip this subscriber
			continue
		default:
			// Subscription still active, continue to delivery
		}

		if Matches(event, sub.filters) {
			// Use a closure with recover to handle the race condition where
			// the channel is closed between the cancel check above and the send below.
			// This is acceptable for async event delivery - the subscriber unsubscribed,
			// so it doesn't need the event anyway.
			func() {
				defer func() {
					// Recover from panic if channel was closed between cancel check and send.
					// This is expected behavior when a subscriber unsubscribes, not an error.
					_ = recover()
				}()

				select {
				case sub.ch <- event:
					delivered++
				case <-sub.cancel:
					// Subscription was cancelled during send attempt, skip
				default:
					// Channel is full (slow consumer)
					dropped++

					if b.config.LogSlowConsumers {
						// Logging happens outside the lock, so slow I/O won't block the API
						logger.Warn("Dropped event due to slow consumer",
							"subscription_id", sub.id,
							"event_type", event.Type,
							"event_id", event.ID)
					}
				}
			}()
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
// This method waits for any in-flight publishAsync goroutines to complete
// before closing the channel to prevent race conditions.
//
// It is safe to call Unsubscribe multiple times with the same ID or
// with an ID that doesn't exist.
func (b *EventBus) Unsubscribe(id string) {
	b.mu.Lock()

	sub, ok := b.subscribers[id]
	if !ok {
		b.mu.Unlock()

		return
	}

	// Remove from map first (while holding lock)
	delete(b.subscribers, id)
	b.metrics.SubscribersTotal.Add(-1)
	b.mu.Unlock()

	// Signal cancellation first (publishAsync will check this)
	close(sub.cancel)

	// Wait for all in-flight publishAsync goroutines to complete.
	// This prevents closing the channel while a goroutine might still send to it.
	b.wg.Wait()

	// Now it's safe to close the channel
	close(sub.ch)

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

// WaitForAsyncPublish waits for all in-flight publishAsync goroutines to complete.
// This is useful for testing to ensure all events have been delivered before
// checking results.
func (b *EventBus) WaitForAsyncPublish() {
	b.wg.Wait()
}
