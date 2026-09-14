// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration for identity/ownership claim
// verification (server/identity resolvers).
package config

import ansconfig "github.com/agntcy/dir/server/identity/ans/config"

// Config holds configuration for record identity/ownership claim verification.
type Config struct {
	// SpiffeTrustDomains maps a SPIFFE trust domain (e.g. "acme.com") to a path
	// containing one or more PEM-encoded CA certificates. A SPIFFE subject
	// whose trust domain has no entry here is verified against its embedded
	// certificate only, without chain validation (dev/unconfigured fallback).
	SpiffeTrustDomains map[string]string `json:"spiffe_trust_domains,omitempty" mapstructure:"spiffe_trust_domains"`

	// Ans configures verification of "ans://" subjects through the Agent Name
	// Service. Off unless enabled; the same block must be configured for the
	// reconciler's identity task.
	Ans ansconfig.Config `json:"ans,omitzero" mapstructure:"ans"`
}
