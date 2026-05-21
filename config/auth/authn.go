// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
)

// Mode specifies the authentication mode (jwt or x509).
type Mode string

const (
	ModeJWT  Mode = "jwt"
	ModeX509 Mode = "x509"
)

// Authn contains configuration for caller authentication. Used by the
// apiserver gRPC ingress, the reconciler regsync credential provider,
// and the daemon (embedded mode).
type Authn struct {
	// Enabled indicates whether authentication is enabled.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// Mode is the authentication mode: "jwt" or "x509".
	Mode Mode `json:"mode,omitempty" mapstructure:"mode"`

	// SocketPath is the SPIFFE workload API socket path.
	SocketPath string `json:"socket_path,omitempty" mapstructure:"socket_path"`

	// Audiences lists the expected audiences for JWT validation
	// (only used in JWT mode).
	Audiences []string `json:"audiences,omitempty" mapstructure:"audiences"`
}

// Validate reports configuration errors. Validation is skipped when
// authentication is disabled.
func (c *Authn) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.SocketPath == "" {
		return errors.New("socket path is required")
	}

	switch c.Mode {
	case ModeJWT:
		if len(c.Audiences) == 0 {
			return errors.New("at least one audience is required for JWT mode")
		}
	case ModeX509:
		// No additional validation required for X.509.
	default:
		return fmt.Errorf("invalid auth mode: %s (must be 'jwt' or 'x509')", c.Mode)
	}

	return nil
}
