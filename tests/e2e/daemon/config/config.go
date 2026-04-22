// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/agntcy/dir/client"

type Config struct {
	// Whether to run runtime tests.
	RunRuntimeDiscoveryTests bool `json:"run_runtime_discovery_tests,omitempty" mapstructure:"run_runtime_discovery_tests"`

	// Client configuration for tests.
	ClientOptions client.Config `json:"client_options" mapstructure:"client_options"`
}
