// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oasf

import (
	"time"

	"github.com/agntcy/dir/client"
)

const (
	// DefaultTimeout is the default OASF resolver timeout.
	DefaultTimeout = 5 * time.Second

	// DefaultLabelKey is the default label key to use for OASF discovery.
	DefaultLabelKey = "org.agntcy/agent-record"
)

// Config holds OASF resolver configuration.
type Config struct {
	// Enabled enables the OASF resolver.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the OASF resolver timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// LabelKey is the label key to use for OASF discovery.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// Runtime configuration for the OASF Client.
	Client *client.Config `json:"client,omitempty" mapstructure:"client"`
}
