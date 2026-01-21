// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package docker

const (
	// DefaultHost is the default Docker daemon socket path.
	DefaultHost = "unix:///var/run/docker.sock"

	// DefaultLabelKey is the default Docker label key to filter containers.
	DefaultLabelKey = "org.agntcy/discover"

	// DefaultLabelValue is the default Docker label value to filter containers.
	DefaultLabelValue = "true"
)

// Config holds Docker runtime configuration.
type Config struct {
	// Host is the Docker daemon socket path.
	Host string `json:"host,omitempty" mapstructure:"host"`

	// LabelKey is the Docker label key to filter containers.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the Docker label value to filter containers.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`
}
