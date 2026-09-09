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
	DefaultRegistryAddress    = "127.0.0.1:5555"
	DefaultRepositoryName     = "dir"
)

type Config struct {
	// Path to a local directory that will be to hold data instead of remote.
	// If this is set to non-empty value, only local store will be used.
	LocalDir string `json:"local_dir,omitempty" mapstructure:"local_dir"`

	// Path to a local directory that will be used to cache metadata.
	// If empty, caching will not be used.
	CacheDir string `json:"cache_dir,omitempty" mapstructure:"cache_dir"`

	// Registry address this server dials for its own store operations.
	// Prefer a private endpoint (e.g. an in-cluster registry service): this
	// value is only handed to remote peers when AdvertisedRegistryAddress is
	// empty.
	RegistryAddress string `json:"registry_address,omitempty" mapstructure:"registry_address"`

	// AdvertisedRegistryAddress is the registry address handed to remote peers
	// via SyncService/RequestRegistryCredentials so they can pull record content
	// from this node. Set it when RegistryAddress is not reachable from outside,
	// which is the usual case for a .svc.cluster.local registry service fronted
	// by a public ingress. Empty falls back to RegistryAddress, preserving the
	// single-address behavior.
	AdvertisedRegistryAddress string `json:"advertised_registry_address,omitempty" mapstructure:"advertised_registry_address"`

	// AdvertisedInsecure is the TLS mode advertised alongside
	// AdvertisedRegistryAddress: true tells peers to use plain HTTP. It is
	// deliberately independent of AuthConfig.Insecure, because the dialed
	// endpoint is commonly plain HTTP in-cluster while the advertised one is
	// HTTPS behind an ingress. Ignored when AdvertisedRegistryAddress is empty,
	// in which case AuthConfig.Insecure is advertised instead, and when that
	// address carries an explicit scheme, which takes precedence.
	AdvertisedInsecure bool `json:"advertised_insecure,omitempty" mapstructure:"advertised_insecure"`

	// Repository name to connect to
	RepositoryName string `json:"repository_name,omitempty" mapstructure:"repository_name"`

	// Authentication configuration
	AuthConfig `json:"auth_config" mapstructure:"auth_config"`
}

// GetRegistryAddress returns the registry address this server dials, with scheme
// and default applied. If RegistryAddress is empty, DefaultRegistryAddress is
// used. When the address has no scheme, http is used for insecure (e.g. E2E,
// internal) and https otherwise.
func (c Config) GetRegistryAddress() (string, error) {
	addr := c.RegistryAddress
	if addr == "" {
		addr = DefaultRegistryAddress
	}

	return addressWithScheme(addr, c.Insecure)
}

// GetAdvertisedRegistryAddress returns the registry address to hand to remote
// peers, with scheme applied. It falls back to GetRegistryAddress when no
// advertised address is configured.
func (c Config) GetAdvertisedRegistryAddress() (string, error) {
	if c.AdvertisedRegistryAddress == "" {
		return c.GetRegistryAddress()
	}

	return addressWithScheme(c.AdvertisedRegistryAddress, c.AdvertisedInsecure)
}

// GetAdvertisedInsecure reports the TLS mode that belongs with
// GetAdvertisedRegistryAddress, so peers are told how to reach the advertised
// endpoint rather than the dialed one.
//
// An explicit scheme on the advertised address wins over AdvertisedInsecure.
// Peers act on this boolean alone - regsync strips the scheme off the address
// before deriving its TLS mode from it - so letting the two disagree would
// silently disable TLS against an https endpoint.
func (c Config) GetAdvertisedInsecure() bool {
	if c.AdvertisedRegistryAddress == "" {
		return c.Insecure
	}

	if scheme, _, found := strings.Cut(c.AdvertisedRegistryAddress, "://"); found {
		return scheme == "http"
	}

	return c.AdvertisedInsecure
}

// addressWithScheme prefixes addr with an explicit scheme when it has none, and
// rejects any scheme other than http or https. Presence of "://" is what marks
// an address as already carrying a scheme, so that e.g. "ftp://host" is rejected
// rather than turned into "https://ftp://host".
func addressWithScheme(addr string, insecure bool) (string, error) {
	if !strings.Contains(addr, "://") {
		if insecure {
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

// GetRepositoryName returns RepositoryName, or DefaultRepositoryName when unset.
func (c Config) GetRepositoryName() string {
	if c.RepositoryName == "" {
		return DefaultRepositoryName
	}

	return c.RepositoryName
}

// GetRepositoryURL returns the full repository URL (registry address + repository name).
func (c Config) GetRepositoryURL() string {
	address := c.RegistryAddress

	if c.RepositoryName != "" {
		return path.Join(address, c.RepositoryName)
	}

	return address
}

// GetAdvertisedRepositoryURL returns the advertised registry address joined with
// the repository name, in the scheme-less form routing publishes as an
// "/oci/<addr>" multiaddr. Empty when no advertised address is configured.
func (c Config) GetAdvertisedRepositoryURL() string {
	if c.AdvertisedRegistryAddress == "" {
		return ""
	}

	// The multiaddr carries a host form, so drop any configured scheme instead of
	// letting path.Join collapse it into "https:/host".
	addr := c.AdvertisedRegistryAddress
	if _, rest, found := strings.Cut(addr, "://"); found {
		addr = rest
	}

	return path.Join(addr, c.GetRepositoryName())
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
