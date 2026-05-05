// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/agntcy/dir/client"

type Config struct {
	// Whether to run rate limit tests.
	RunRateLimitTests bool `json:"run_rate_limit_tests,omitempty" mapstructure:"run_rate_limit_tests"`

	// Client configuration for tests.
	ClientOptions client.Config `json:"client_options" mapstructure:"client_options"`
}
