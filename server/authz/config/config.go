// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

// Config contains configuration for AuthZ services.
type Config struct {
	// Spiffe socket path
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	// Spiffe trust domain
	TrustDomain string `json:"trust_domain,omitempty" mapstructure:"trust_domain"`
}
