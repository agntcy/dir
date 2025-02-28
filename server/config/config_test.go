// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

//nolint:testifylint
package config

import (
	"testing"

	fsconfig "github.com/agntcy/dir/server/store/localfs/config"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	tests := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedConfig *Config
	}{
		{
			Name: "Custom config",
			EnvVars: map[string]string{
				"DIRECTORY_SERVER_LISTEN_ADDRESS":                "example.com:8889",
				"DIRECTORY_SERVER_HEALTHCHECK_ADDRESS":           "example.com:18888",
				"DIRECTORY_SERVER_PROVIDER":                      "provider",
				"DIRECTORY_SERVER_LOCALFS_DIR":                   "local-dir-fs",
				"DIRECTORY_SERVER_OCI_LOCAL_DIR":                 "local-dir",
				"DIRECTORY_SERVER_OCI_REGISTRY_ADDRESS":          "example.com:5001",
				"DIRECTORY_SERVER_OCI_REPOSITORY_NAME":           "test-dir",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_INSECURE":      "true",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_USERNAME":      "username",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_PASSWORD":      "password",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_ACCESS_TOKEN":  "access-token",
				"DIRECTORY_SERVER_OCI_AUTH_CONFIG_REFRESH_TOKEN": "refresh-token",
			},
			ExpectedConfig: &Config{
				ListenAddress:      "example.com:8889",
				HealthCheckAddress: "example.com:18888",
				Provider:           "provider",
				LocalFS: fsconfig.Config{
					Dir: "local-dir-fs",
				},
				OCI: ociconfig.Config{
					LocalDir:        "local-dir",
					RegistryAddress: "example.com:5001",
					RepositoryName:  "test-dir",
					AuthConfig: ociconfig.AuthConfig{
						Insecure:     true,
						Username:     "username",
						Password:     "password",
						RefreshToken: "refresh-token",
						AccessToken:  "access-token",
					},
				},
			},
		},
		{
			Name:    "Default config",
			EnvVars: map[string]string{},
			ExpectedConfig: &Config{
				ListenAddress:      DefaultListenAddress,
				HealthCheckAddress: DefaultHealthCheckAddress,
				Provider:           DefaultProvider,
				LocalFS: fsconfig.Config{
					Dir: fsconfig.DefaultDir,
				},
				OCI: ociconfig.Config{
					RegistryAddress: ociconfig.DefaultRegistryAddress,
					RepositoryName:  ociconfig.DefaultRepositoryName,
					AuthConfig: ociconfig.AuthConfig{
						Insecure: ociconfig.DefaultAuthConfigInsecure,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			for k, v := range test.EnvVars {
				t.Setenv(k, v)
			}

			config, err := LoadConfig()
			assert.NoError(t, err)
			assert.Equal(t, *config, *test.ExpectedConfig)
		})
	}
}
