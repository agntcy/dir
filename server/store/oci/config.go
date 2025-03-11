// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package oci

const (
	DefaultRegistryAddress = "127.0.0.1:5000"
	DefaultRepositoryName  = "dir"
)

type Config struct {
	// Registry address to connect to
	RegistryAddress string `json:"oci_registry_address,omitempty" mapstructure:"oci_registry_address"`

	// Repository name to connect to
	RepositoryName string `json:"oci_repository_name,omitempty" mapstructure:"oci_repository_name"`

	// Authentication configuration
	AuthConfig `json:"auth_config,omitempty" mapstructure:"auth_config"`
}

// AuthConfig represents the configuration for authentication.
type AuthConfig struct {
	Username string `json:"username,omitempty" mapstructure:"username"`
	Password string `json:"password,omitempty" mapstructure:"password"`
}
