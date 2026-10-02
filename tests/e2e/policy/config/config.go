// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/agntcy/dir/client"

// Config configures the content-policy enforcement suite.
type Config struct {
	// Client configuration for tests.
	ClientOptions client.Config `json:"client_options" mapstructure:"client_options"`

	// DaemonConfig is the daemon's configuration file, relative to the suite's
	// directory, and DaemonDataDir its data directory. A dry run reads the
	// node's records from them.
	DaemonConfig  string `json:"daemon_config" mapstructure:"daemon_config"`
	DaemonDataDir string `json:"daemon_data_dir" mapstructure:"daemon_data_dir"`
}
