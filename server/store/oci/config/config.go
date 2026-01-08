// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

// RegistryType defines the type of OCI registry backend.
type RegistryType string

const (
	// RegistryTypeZot uses Zot registry.
	RegistryTypeZot RegistryType = "zot"

	// RegistryTypeGeneric uses standard OCI registry.
	RegistryTypeGeneric RegistryType = "generic"

	// DefaultRegistryType is the default registry type for backward compatibility.
	DefaultRegistryType = RegistryTypeZot
)

const (
	DefaultAuthConfigInsecure = true
	DefaultRegistryAddress    = "127.0.0.1:5000"
	DefaultRepositoryName     = "dir"
)

type Config struct {
	// Type specifies the registry type (zot, generic).
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
