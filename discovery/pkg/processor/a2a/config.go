package a2a

import "time"

const DefaultDiscoveryPaths string = "/.well-known/agent-card.json,/.well-known/card.json"

type Config struct {
	// Enabled enables the A2A discovery processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the A2A discovery processor timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// Paths is the list of paths to probe.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
}
