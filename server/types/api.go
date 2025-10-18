// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/agntcy/dir/server/config"
	"github.com/agntcy/dir/server/events"
)

// TODO: Extend with cleaning and garbage collection support.
type API interface {
	// Options returns API options
	Options() APIOptions

	// Store returns an implementation of the StoreAPI
	Store() StoreAPI

	// Routing returns an implementation of the RoutingAPI
	Routing() RoutingAPI

	// Database returns an implementation of the DatabaseAPI
	Database() DatabaseAPI
}

// APIOptions collects internal dependencies for all API services.
type APIOptions interface {
	// Config returns the config data. Read only! Unsafe to edit.
	Config() *config.Config

	// EventBus returns the safe event bus for publishing events.
	// Returns a nil-safe wrapper that won't panic even if events are disabled.
	EventBus() *events.SafeEventBus

	// WithEventBus returns a new APIOptions with the event bus set.
	WithEventBus(bus *events.SafeEventBus) APIOptions
}

type options struct {
	config   *config.Config
	eventBus *events.SafeEventBus
}

func NewOptions(config *config.Config) APIOptions {
	return &options{
		config:   config,
		eventBus: events.NewSafeEventBus(nil), // Default to nil-safe no-op
	}
}

func (o *options) Config() *config.Config { return o.config }

func (o *options) EventBus() *events.SafeEventBus { return o.eventBus }

func (o *options) WithEventBus(bus *events.SafeEventBus) APIOptions {
	return &options{
		config:   o.config,
		eventBus: bus,
	}
}
