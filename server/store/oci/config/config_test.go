// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetRegistryAddressUsesDefaultRegistryPort(t *testing.T) {
	address, err := Config{
		Insecure: true,
	}.GetRegistryAddress()

	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:5555", address)
}

// The advertised address falls back to the dialed one, which is what keeps
// single-address deployments (and any config written before the split) working.
func TestGetAdvertisedRegistryAddressFallsBackToDialed(t *testing.T) {
	cfg := Config{
		RegistryAddress: "dir-zot.dir.svc.cluster.local:5000",
		Insecure:        true,
	}

	address, err := cfg.GetAdvertisedRegistryAddress()

	require.NoError(t, err)
	require.Equal(t, "http://dir-zot.dir.svc.cluster.local:5000", address)
	require.True(t, cfg.GetAdvertisedInsecure())
}

// The advertised endpoint carries its own TLS mode: in-cluster registries are
// commonly plain HTTP while the public endpoint they are fronted by is HTTPS.
func TestGetAdvertisedRegistryAddressUsesOwnTLSMode(t *testing.T) {
	cfg := Config{
		RegistryAddress:           "dir-zot.dir.svc.cluster.local:5000",
		AdvertisedRegistryAddress: "store.example.com",
		Insecure:                  true,
	}

	dialed, err := cfg.GetRegistryAddress()
	require.NoError(t, err)
	require.Equal(t, "http://dir-zot.dir.svc.cluster.local:5000", dialed)

	advertised, err := cfg.GetAdvertisedRegistryAddress()
	require.NoError(t, err)
	require.Equal(t, "https://store.example.com", advertised)
	require.False(t, cfg.GetAdvertisedInsecure())
}

func TestGetAdvertisedRegistryAddressInsecure(t *testing.T) {
	cfg := Config{
		RegistryAddress:           "dir-zot.dir.svc.cluster.local:5000",
		AdvertisedRegistryAddress: "store.example.com:5000",
		AdvertisedInsecure:        true,
	}

	advertised, err := cfg.GetAdvertisedRegistryAddress()

	require.NoError(t, err)
	require.Equal(t, "http://store.example.com:5000", advertised)
	require.True(t, cfg.GetAdvertisedInsecure())
}

func TestGetAdvertisedRegistryAddressKeepsExplicitScheme(t *testing.T) {
	cfg := Config{AdvertisedRegistryAddress: "https://store.example.com"}

	advertised, err := cfg.GetAdvertisedRegistryAddress()

	require.NoError(t, err)
	require.Equal(t, "https://store.example.com", advertised)
	require.False(t, cfg.GetAdvertisedInsecure())
}

// Peers receive the address and the TLS mode as separate values and act on the
// boolean, so the two must not be able to disagree: an explicit scheme decides.
func TestGetAdvertisedInsecureFollowsExplicitScheme(t *testing.T) {
	httpsAddr := Config{
		AdvertisedRegistryAddress: "https://store.example.com",
		AdvertisedInsecure:        true,
	}
	require.False(t, httpsAddr.GetAdvertisedInsecure())

	httpAddr := Config{
		AdvertisedRegistryAddress: "http://dir-zot.dir.svc.cluster.local:5000",
		AdvertisedInsecure:        false,
	}
	require.True(t, httpAddr.GetAdvertisedInsecure())
}

func TestGetAdvertisedRegistryAddressRejectsBadScheme(t *testing.T) {
	_, err := Config{
		AdvertisedRegistryAddress: "ftp://store.example.com",
	}.GetAdvertisedRegistryAddress()

	require.Error(t, err)
}

func TestGetAdvertisedRepositoryURL(t *testing.T) {
	tests := []struct {
		name     string
		cfg      Config
		expected string
	}{
		{
			name:     "empty when no advertised address",
			cfg:      Config{RegistryAddress: "dir-zot.dir.svc.cluster.local:5000", RepositoryName: "dir"},
			expected: "",
		},
		{
			name:     "joins advertised address and repository",
			cfg:      Config{AdvertisedRegistryAddress: "ghcr.io/org", RepositoryName: "agents"},
			expected: "ghcr.io/org/agents",
		},
		{
			name:     "applies default repository name",
			cfg:      Config{AdvertisedRegistryAddress: "store.example.com"},
			expected: "store.example.com/" + DefaultRepositoryName,
		},
		{
			name:     "drops scheme so the host form stays intact",
			cfg:      Config{AdvertisedRegistryAddress: "https://store.example.com", RepositoryName: "dir"},
			expected: "store.example.com/dir",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.cfg.GetAdvertisedRepositoryURL())
		})
	}
}

func TestGetRepositoryName(t *testing.T) {
	require.Equal(t, DefaultRepositoryName, Config{}.GetRepositoryName())
	require.Equal(t, "custom", Config{RepositoryName: "custom"}.GetRepositoryName())
}
