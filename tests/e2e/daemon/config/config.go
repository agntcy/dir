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

	// Client configuration for tests.
	ClientOptions client.Config `json:"client_options" mapstructure:"client_options"`
}
