// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"github.com/agntcy/dir/runtime/discovery/runtime/docker"
	"github.com/agntcy/dir/runtime/discovery/runtime/k8s"
	"github.com/agntcy/dir/runtime/discovery/types"
)

// Config holds all runtime-specific configuration.
type Config struct {
	// Runtime type to use.
	Type types.RuntimeType `json:"type" mapstructure:"type"`

	// Docker runtime configuration.
	Docker docker.Config `json:"docker" mapstructure:"docker"`

	// Kubernetes runtime configuration.
	Kubernetes k8s.Config `json:"kubernetes" mapstructure:"kubernetes"`
}
