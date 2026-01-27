package openapi

import "time"

type Config struct {
	// Enabled enables the OpenAPI discovery processor.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Timeout is the OpenAPI fetch timeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// Paths is the list of OpenAPI spec paths to check.
	Paths []string `json:"paths,omitempty" mapstructure:"paths"`
}
