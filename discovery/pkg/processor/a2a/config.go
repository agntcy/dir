package a2a

import "time"

const (
	DefaultDiscoveryPaths = "/.well-known/agent-card.json,/.well-known/card.json"
	DefaultLabelKey       = "org.agntcy/type"
	DefaultLabelValue     = "a2a"
)

type Config struct {
	// Enabled enables the A2A discovery processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the A2A discovery processor timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// LabelKey is the label key to use for A2A discovery.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`

	// LabelValue is the label value to use for A2A discovery.
	LabelValue string `json:"label_value,omitempty" mapstructure:"label_value"`

	// Paths is the list of paths to probe.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
}
