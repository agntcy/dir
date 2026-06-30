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
	DefaultRegistryAuthInsecure = true
	DefaultRegistryAddress      = "127.0.0.1:5555"
	DefaultRepositoryName       = "dir"
)

// Registry is the OCI registry connection configuration shared by both the
// apiserver (primary writer/reader) and the reconciler (reader, regsync writer).
// Declaring it once at the top level of Config eliminates the duplication
// between server.store and reconciler.local_registry.
type Registry struct {
	// LocalDir, when non-empty, points to a local directory used instead of a
	// remote registry (the daemon hosts an embedded Zot instance here).
	LocalDir string `json:"local_dir,omitempty" mapstructure:"local_dir"`

	// CacheDir is a local directory used to cache metadata. Empty disables caching.
	CacheDir string `json:"cache_dir,omitempty" mapstructure:"cache_dir"`

	// RegistryAddress is the address of the OCI registry (host:port or scheme://host:port).
	RegistryAddress string `json:"registry_address,omitempty" mapstructure:"registry_address"`

	// RepositoryName is the repository path within the registry.
	RepositoryName string `json:"repository_name,omitempty" mapstructure:"repository_name"`

	// RegistryAuth holds the credentials used to talk to the registry.
	RegistryAuth `json:"auth_config" mapstructure:"auth_config"`
}

// GetRegistryAddress returns the registry address with scheme applied.
// If RegistryAddress is empty, DefaultRegistryAddress is used.
// When the address has no scheme, http is used for insecure registries, https otherwise.
func (c Registry) GetRegistryAddress() (string, error) {
	addr := c.RegistryAddress
	if addr == "" {
		addr = DefaultRegistryAddress
	}

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

// GetRepositoryURL returns the registry address joined with the repository name,
// without scheme. Used by regclient/regsync which strips the scheme itself.
func (c Registry) GetRepositoryURL() string {
	address := c.RegistryAddress

	if c.RepositoryName != "" {
		return path.Join(address, c.RepositoryName)
	}

	return address
}

// RegistryAuth holds OCI registry credentials.
//
//nolint:gosec // G117: intentional config field for OCI auth
type RegistryAuth struct {
	// Insecure marks the registry as plain HTTP.
	Insecure bool `json:"insecure" mapstructure:"insecure"`

	Username string `json:"username,omitempty" mapstructure:"username"`

	Password string `json:"password,omitempty" mapstructure:"password"`

	RefreshToken string `json:"refresh_token,omitempty" mapstructure:"refresh_token"`

	AccessToken string `json:"access_token,omitempty" mapstructure:"access_token"`
}
