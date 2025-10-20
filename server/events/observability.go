// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package events

import "sync/atomic"

// Metrics holds simple counters for event system observability.
// All counters use atomic operations for thread-safety.
type Metrics struct {
	// PublishedTotal is the total number of events published to the bus
	PublishedTotal atomic.Uint64

	// DeliveredTotal is the total number of events delivered to subscribers
	DeliveredTotal atomic.Uint64

	// DroppedTotal is the total number of events dropped due to slow consumers
	DroppedTotal atomic.Uint64

	// SubscribersTotal is the current number of active subscribers
	// This can be negative temporarily during concurrent operations, but will stabilize
	SubscribersTotal atomic.Int64
}

// MetricsSnapshot is a point-in-time snapshot of metrics values.
// Unlike Metrics, this is safe to copy and serialize.
type MetricsSnapshot struct {
	PublishedTotal   uint64
	DeliveredTotal   uint64
	DroppedTotal     uint64
	SubscribersTotal int64
}
