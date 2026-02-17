// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

const (
	DefaultAuthConfigInsecure = true
	DefaultRegistryAddress    = "127.0.0.1:5000"
	DefaultRepositoryName     = "dir"
)

type Config struct {
	// Path to a local directory that will be to hold data instead of remote.
	// If this is set to non-empty value, only local store will be used.
	LocalDir string `json:"local_dir,omitempty" mapstructure:"local_dir"`

	// Path to a local directory that will be used to cache metadata.
	// If empty, caching will not be used.
	CacheDir string `json:"cache_dir,omitempty" mapstructure:"cache_dir"`

	// Registry address to connect to
	RegistryAddress string `json:"registry_address,omitempty" mapstructure:"registry_address"`

	// Repository name to connect to
	RepositoryName string `json:"repository_name,omitempty" mapstructure:"repository_name"`

	// Authentication configuration
	AuthConfig `json:"auth_config" mapstructure:"auth_config"`
}

// GetRegistryAddress returns the registry address with scheme and default applied.
// If RegistryAddress is empty, DefaultRegistryAddress is used. When the address
// has no scheme, http is used for insecure (e.g. E2E, internal) and https otherwise.
func (c Config) GetRegistryAddress() (string, error) {
	addr := c.RegistryAddress
	if addr == "" {
		addr = DefaultRegistryAddress
	}

	// Add explicit scheme when none is present
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		if c.Insecure {
			addr = "http://" + addr
		} else {
			addr = "https://" + addr
		}
	}

	parsed, err := url.Parse(addr)
	if err != nil {
		return "", fmt.Errorf("invalid registry address: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("registry address must use http or https scheme")
	}

	return addr, nil
}

// GetRepositoryURL returns the full repository URL (registry address + repository name).
func (c Config) GetRepositoryURL() string {
	address := c.RegistryAddress

	if c.RepositoryName != "" {
		return path.Join(address, c.RepositoryName)
	}

	return address
}

// AuthConfig represents the configuration for authentication.
type AuthConfig struct {
	Insecure bool `json:"insecure" mapstructure:"insecure"`

	Username string `json:"username,omitempty" mapstructure:"username"`

	Password string `json:"password,omitempty" mapstructure:"password"`

	RefreshToken string `json:"refresh_token,omitempty" mapstructure:"refresh_token"`

	AccessToken string `json:"access_token,omitempty" mapstructure:"access_token"`
}
