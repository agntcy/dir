// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	oci "github.com/agntcy/dir/server/store/oci/config"
)

const (
	DefaultProvider = "oci"

	// DefaultVerificationEnabled controls whether name verification is enabled.
	DefaultVerificationEnabled = true
)

type Config struct {
	// Provider is the type of the storage provider.
	Provider string `json:"c,omitempty" mapstructure:"provider"`

	// Config for OCI database.
	OCI oci.Config `json:"oci,omitempty" mapstructure:"oci"`

	// Verification configures name ownership verification.
	Verification VerificationConfig `json:"verification,omitempty" mapstructure:"verification"`
}

// VerificationConfig defines name verification configuration.
type VerificationConfig struct {
	// Enabled controls whether name verification is performed.
	// Default: true
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`
}
