// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The only thing left to validate about enforcement is how often it is
// rechecked; which policies apply is said by the operation policies.
func TestConfigRefreshInterval(t *testing.T) {
	t.Parallel()

	require.NoError(t, (&Config{}).Validate())
	require.EqualError(t, (&Config{RefreshInterval: -time.Second}).Validate(),
		"policy: refresh_interval must not be negative")

	require.Equal(t, DefaultRefreshInterval, (&Config{}).GetRefreshInterval())
	require.Equal(t, time.Minute, (&Config{RefreshInterval: time.Minute}).GetRefreshInterval())

	// Validate refuses a negative, but nothing on the serving path calls it,
	// so the getter has to fall back rather than pass one on: the resolver
	// reads this as an age to compare against, and negative makes every
	// enforcement it holds stale on arrival.
	require.Equal(t, DefaultRefreshInterval, (&Config{RefreshInterval: -time.Second}).GetRefreshInterval())
}
