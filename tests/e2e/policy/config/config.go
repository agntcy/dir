// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/agntcy/dir/client"

// Config configures the content-policy enforcement suite.
type Config struct {
	// Client configuration for tests.
	ClientOptions client.Config `json:"client_options" mapstructure:"client_options"`
}
