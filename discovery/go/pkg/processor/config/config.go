package config

import (
	"github.com/agntcy/dir/discovery/pkg/processor/health"
	"github.com/agntcy/dir/discovery/pkg/processor/openapi"
)

// Config holds all processor-specific configuration.
type Config struct {
	// Processor workers count.
	Workers int `json:"workers" mapstructure:"workers"`

	// Health processor configuration.
	Health health.Config `json:"health" mapstructure:"health"`

	// OpenAPI processor configuration.
	OpenAPI openapi.Config `json:"openapi" mapstructure:"openapi"`
}
