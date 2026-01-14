// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/agntcy/dir/utils/logging"

var logger = logging.Logger("store/oci/config")

// RegistryType defines the type of OCI registry backend.
// Only explicitly tested registries are supported.
type RegistryType string

const (
	// RegistryTypeZot uses Zot registry.
	RegistryTypeZot RegistryType = "zot"

	// RegistryTypeGHCR uses GitHub Container Registry.
	RegistryTypeGHCR RegistryType = "ghcr"

	// RegistryTypeDockerHub uses Docker Hub registry.
	RegistryTypeDockerHub RegistryType = "dockerhub"

	// DefaultRegistryType is the default registry type for backward compatibility.
	DefaultRegistryType = RegistryTypeZot
)

// IsSupported returns true if the registry type is explicitly supported and tested.
// Logs a warning if an experimental registry type (ghcr, dockerhub) is used.
func (r RegistryType) IsSupported() bool {
	switch r {
	case RegistryTypeZot:
		return true
	case RegistryTypeGHCR, RegistryTypeDockerHub:
		logger.Warn("Registry type support is experimental and not fully tested. "+
			"The default deployment configuration (Zot registry) is not appropriate for this registry type. "+
			"Do not use in production.",
			"registry_type", string(r))

		return true
	default:
		return false
	}
}

const (
	DefaultAuthConfigInsecure = true
	DefaultRegistryAddress    = "127.0.0.1:5000"
	DefaultRepositoryName     = "dir"
)

type Config struct {
	// Type specifies the registry type (zot, ghcr, dockerhub).
	// Defaults to "zot" for backward compatibility.
	Type RegistryType `json:"type,omitempty" mapstructure:"type"`

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
	AuthConfig `json:"auth_config,omitempty" mapstructure:"auth_config"`
}

// GetType returns the registry type, defaulting to Zot if not specified.
func (c Config) GetType() RegistryType {
	if c.Type == "" {
		return DefaultRegistryType
	}

	return c.Type
}

// AuthConfig represents the configuration for authentication.
type AuthConfig struct {
	Insecure bool `json:"insecure" mapstructure:"insecure"`

	Username string `json:"username,omitempty" mapstructure:"username"`

	Password string `json:"password,omitempty" mapstructure:"password"`

	RefreshToken string `json:"refresh_token,omitempty" mapstructure:"refresh_token"`

	AccessToken string `json:"access_token,omitempty" mapstructure:"access_token"`
}
