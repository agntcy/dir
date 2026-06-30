// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	"github.com/agntcy/dir/config"
	"github.com/stretchr/testify/require"
)

func TestGetRegistryAddressUsesDefaultRegistryPort(t *testing.T) {
	address, err := config.Registry{
		RegistryAuth: config.RegistryAuth{
			Insecure: true,
		},
	}.GetRegistryAddress()

	require.NoError(t, err)
	require.Equal(t, "http://127.0.0.1:5555", address)
}
