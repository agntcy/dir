package config

import (
	"github.com/agntcy/dir/discovery/pkg/processor/a2a"
	"github.com/agntcy/dir/discovery/pkg/processor/oasf"
)

// Config holds all processor-specific configuration.
type Config struct {
	// Processor workers count.
	Workers int `json:"workers" mapstructure:"workers"`

	// A2A processor configuration.
	A2A a2a.Config `json:"a2a" mapstructure:"a2a"`

	// OASF processor configuration.
	OASF oasf.Config `json:"oasf" mapstructure:"oasf"`
}
