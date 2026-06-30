// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"errors"
	"fmt"
)

// Mode specifies the authentication mode.
type Mode string

const (
	ModeJWT  Mode = "jwt"
	ModeX509 Mode = "x509"
)

// Authn configures caller authentication for the apiserver gRPC ingress
// and the reconciler regsync credential provider.
type Authn struct {
	Enabled    bool     `json:"enabled,omitempty"     mapstructure:"enabled"`
	Mode       Mode     `json:"mode,omitempty"        mapstructure:"mode"`
	SocketPath string   `json:"socket_path,omitempty" mapstructure:"socket_path"`
	Audiences  []string `json:"audiences,omitempty"   mapstructure:"audiences"`
}

// Validate reports configuration errors. Skipped when authentication is disabled.
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
		// no additional fields required
	default:
		return fmt.Errorf("invalid auth mode: %s (must be 'jwt' or 'x509')", c.Mode)
	}

	return nil
}
