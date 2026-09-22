// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/agntcy/dir/client"

type Config struct {
	// Whether to run runtime tests.
	RunRuntimeDiscoveryTests bool `json:"run_runtime_discovery_tests,omitempty" mapstructure:"run_runtime_discovery_tests"`

	// Compose file holding the workload the runtime discovery tests start.
	// Relative paths resolve against the test package directory, which is the
	// working directory `go test -C` runs in. Only read when
	// RunRuntimeDiscoveryTests is set: the testenvs share this package and
	// only the external one defines such a workload.
	RuntimeWorkloadComposeFile string `json:"runtime_workload_compose_file,omitempty" mapstructure:"runtime_workload_compose_file"`

	// GatewayAddress is the base URL of the daemon's HTTP gateway (e.g.
	// "http://localhost:18236"). Empty when the gateway is not enabled in this
	// environment, in which case the gateway specs skip.
	GatewayAddress string `json:"gateway_address,omitempty" mapstructure:"gateway_address"`

	// ExpectNoExtractor reports that this environment deliberately runs the
	// gateway with no OASF extractor resolvable, so the extractor-backed
	// endpoints must answer 503 rather than serve. It is a positive assertion,
	// not a skip: the point of the environment is that a node deployed without
	// an extractor still starts and degrades only those routes.
	ExpectNoExtractor bool `json:"expect_no_extractor,omitempty" mapstructure:"expect_no_extractor"`

	// Client configuration for tests.
	ClientOptions client.Config `json:"client_options" mapstructure:"client_options"`
}
