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
	// Create a local registry if true
	CreateLocalRegistry bool `json:"create_local_registry,omitempty" mapstructure:"create_local_registry"`

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
	address := c.RegistryAddress
	// Add explicit scheme when none is present
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		if c.Insecure {
			address = "http://" + address
		} else {
			address = "https://" + address
		}
	}

	parsed, err := url.Parse(address)
	if err != nil {
		return "", fmt.Errorf("invalid registry address: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("registry address must use http or https scheme")
	}

	return address, nil
}

// GetRepositoryURL returns the full repository URL (registry address + repository name).
func (c Config) GetRepositoryURL() string {
	if c.RepositoryName != "" {
		return path.Join(c.RegistryAddress, c.RepositoryName)
	}

	return c.RegistryAddress
}

// AuthConfig represents the configuration for authentication.
//
//nolint:gosec // G117: intentional config field for OCI auth
type AuthConfig struct {
	Insecure bool `json:"insecure" mapstructure:"insecure"`

	Username string `json:"username,omitempty" mapstructure:"username"`

	Password string `json:"password,omitempty" mapstructure:"password"`

	RefreshToken string `json:"refresh_token,omitempty" mapstructure:"refresh_token"`

	AccessToken string `json:"access_token,omitempty" mapstructure:"access_token"`
}
