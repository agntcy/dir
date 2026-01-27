package config

import (
	"github.com/agntcy/dir/discovery/pkg/processor/a2a"
)

// Config holds all processor-specific configuration.
type Config struct {
	// Processor workers count.
	Workers int `json:"workers" mapstructure:"workers"`

	// Health processor configuration.
	A2A a2a.Config `json:"a2a" mapstructure:"a2a"`
}
