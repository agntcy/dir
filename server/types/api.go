// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"github.com/agntcy/dir/server/config"
)

// TODO: Extend with cleaning and garbage collection support.
type API interface {
	// Options returns API options
	Options() APIOptions

	// Store returns an implementation of the StoreAPI
	Store() StoreAPI

	// Routing returns an implementation of the RoutingAPI
	Routing() RoutingAPI
}

// APIOptions collects internal dependencies for all API services.
type APIOptions interface {
	// Config returns the config data. Read only! Unsafe to edit.
	Config() *config.Config
}

type options struct {
	config *config.Config
}

func NewOptions(config *config.Config) APIOptions {
	return &options{
		config: config,
	}
}

func (o options) Config() *config.Config { return o.config }
