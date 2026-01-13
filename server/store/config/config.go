// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	oci "github.com/agntcy/dir/server/store/oci/config"
)

const (
	DefaultProvider = "oci"

	// DefaultVerificationEnabled controls whether domain verification is enabled.
	DefaultVerificationEnabled = true
)

type Config struct {
	// Provider is the type of the storage provider.
	Provider string `json:"c,omitempty" mapstructure:"provider"`

	// Config for OCI database.
	OCI oci.Config `json:"oci,omitempty" mapstructure:"oci"`

	// Verification configures domain ownership verification.
	Verification VerificationConfig `json:"verification,omitempty" mapstructure:"verification"`
}

// VerificationConfig defines domain verification configuration.
type VerificationConfig struct {
	// Enabled controls whether domain verification is performed.
	// Default: true
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// AllowInsecure allows HTTP (instead of HTTPS) for well-known file fetching.
	// WARNING: Only use for local development/testing. Never enable in production.
	// Default: false
	AllowInsecure bool `json:"allow_insecure,omitempty" mapstructure:"allow_insecure"`
}
