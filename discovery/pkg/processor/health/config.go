package health

import "time"

type Config struct {
	// Enabled enables the health check processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the health check timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// Paths is the list of health check paths to probe.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
}
