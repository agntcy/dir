// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration for the naming service.
package config

// Config holds configuration for the naming service.
type Config struct {
	// Enabled enables name verification.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`
}
