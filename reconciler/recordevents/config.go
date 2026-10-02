// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package recordevents tells the reconciler when records arrive on the server,
// so the tasks that act on new records need not wait for their next interval.
package recordevents

import "time"

const (
	// DefaultWindow is how long arrivals are gathered before the indexer is
	// woken.
	DefaultWindow = 2 * time.Second

	// DefaultReconnectDelay is how long to wait before listening again after
	// the event stream ends.
	DefaultReconnectDelay = 5 * time.Second
)

// Config configures how the reconciler reacts to records arriving on the
// server. The tasks' own intervals remain the backstop: they find whatever an
// event did not announce, such as records copied in by registry sync.
type Config struct {
	// Enabled turns the reaction on. It needs a source of events: the
	// apiserver's address in a standalone reconciler, the server itself in the
	// daemon.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Window is how long arrivals are gathered before the indexer is woken. The
	// first arrival opens a window and the others that come before it closes
	// share its run, so a burst of pushes costs one run of the indexer, not one
	// each, and no record waits longer than the window.
	Window time.Duration `json:"window,omitempty" mapstructure:"window"`

	// ReconnectDelay is how long to wait before listening again after the event
	// stream ends.
	ReconnectDelay time.Duration `json:"reconnect_delay,omitempty" mapstructure:"reconnect_delay"`
}

// GetWindow returns the window, or the default when it is not set.
func (c Config) GetWindow() time.Duration {
	if c.Window <= 0 {
		return DefaultWindow
	}

	return c.Window
}

// GetReconnectDelay returns the reconnect delay, or the default when it is not
// set.
func (c Config) GetReconnectDelay() time.Duration {
	if c.ReconnectDelay <= 0 {
		return DefaultReconnectDelay
	}

	return c.ReconnectDelay
}
