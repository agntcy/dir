// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Config{}).Validate())
	require.NoError(t, (&Config{Enabled: true, Dir: "/etc/agntcy/dir/policies"}).Validate())
	require.EqualError(t, (&Config{Enabled: true}).Validate(), "policy directory is required")
}
