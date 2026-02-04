// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package a2a

import "time"

const (
	// DefaultDiscoveryPaths is the default list of paths to probe for A2A discovery.
	DefaultDiscoveryPaths = "/.well-known/agent-card.json,/.well-known/card.json"

	// DefaultTimeout is the default timeout for A2A discovery.
	DefaultTimeout = 5 * time.Second

	// DefaultLabelKey is the default label key for A2A discovery.
	DefaultLabelKey = "org.agntcy/agent-type"

	// DefaultLabelValue is the default label value for A2A discovery.
	DefaultLabelValue = "a2a"
)

type Config struct {
	// Enabled enables the A2A resolver.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the A2A resolver timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// LabelKey is the label key to use for A2A discovery.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the label value to use for A2A discovery.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`

	// Paths is the list of paths to probe.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
}
