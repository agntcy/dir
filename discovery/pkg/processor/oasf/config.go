package oasf

import "time"

const (
	DefaultLabelKey = "org.agntcy/oasf"
)

type Config struct {
	// Enabled enables the OASF discovery processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the OASF discovery processor timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// LabelKey is the label key to use for OASF discovery.
	LabelKey string `json:"label_key,omitempty" mapstructure:"label_key"`
}
