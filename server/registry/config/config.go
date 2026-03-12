// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

const (
	// DefaultRegistryListenAddress is the default address for the OCI registry server.
	DefaultRegistryListenAddress = "0.0.0.0:8333"
)

type Config struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`

	ListenAddress string `mapstructure:"listen_address" yaml:"listen_address"`
}
