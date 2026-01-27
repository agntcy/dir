package config

import (
	"github.com/agntcy/dir/discovery/pkg/runtime/docker"
	"github.com/agntcy/dir/discovery/pkg/runtime/k8s"
	"github.com/agntcy/dir/discovery/pkg/types"
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
