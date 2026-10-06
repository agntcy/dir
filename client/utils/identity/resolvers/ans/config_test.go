// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"slices"
	"testing"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRootKeyLine = "example-log+1a2b3c4d+AjBZ"

func TestConfigDefaults(t *testing.T) {
	var cfg Config

	assert.Equal(t, DefaultTimeout, cfg.GetTimeout())
	assert.Equal(t, DefaultRootKeysTTL, cfg.GetRootKeysTTL())
	assert.Equal(t, DefaultStatusCacheTTL, cfg.GetStatusCacheTTL())
	assert.Equal(t, DefaultClockSkew, cfg.GetClockSkew())

	cfg = Config{Timeout: time.Second, RootKeysTTL: time.Minute, StatusCacheTTL: 2 * time.Second, ClockSkew: 3 * time.Second}

	assert.Equal(t, time.Second, cfg.GetTimeout())
	assert.Equal(t, time.Minute, cfg.GetRootKeysTTL())
	assert.Equal(t, 2*time.Second, cfg.GetStatusCacheTTL())
	assert.Equal(t, 3*time.Second, cfg.GetClockSkew())
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:   "pinned",
			config: Config{TrustedLogHosts: []string{" Log.Example.com:443 ", "localhost:18443", "log.example.com"}, RootKeys: []string{" " + testRootKeyLine + " ", ""}},
		},
		{
			name:   "unpinned",
			config: Config{TrustedLogHosts: []string{testLogHost}, AllowUnpinnedRootKeys: true},
		},
		{
			name:   "durations at their bounds",
			config: Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, ClockSkew: scitt.MaxClockSkew},
		},
		{
			name:    "no hosts",
			config:  Config{RootKeys: []string{testRootKeyLine}},
			wantErr: "trusted_log_hosts must list at least one",
		},
		{
			name:    "empty host entry",
			config:  Config{TrustedLogHosts: []string{testLogHost, " "}, RootKeys: []string{testRootKeyLine}},
			wantErr: "contains an empty entry",
		},
		{
			name:    "host with scheme",
			config:  Config{TrustedLogHosts: []string{"https://log.example.com"}, RootKeys: []string{testRootKeyLine}},
			wantErr: "must not contain a scheme or path",
		},
		{
			name:    "host without a name",
			config:  Config{TrustedLogHosts: []string{":443"}, RootKeys: []string{testRootKeyLine}},
			wantErr: "has no host name",
		},
		{
			name:    "no root keys without opt-in",
			config:  Config{TrustedLogHosts: []string{testLogHost}},
			wantErr: "root_keys is empty",
		},
		{
			name:    "whitespace-only root keys count as none",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{" ", ""}},
			wantErr: "root_keys is empty",
		},
		{
			name:    "root keys and opt-in together",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, AllowUnpinnedRootKeys: true},
			wantErr: "cannot both be set",
		},
		{
			name:    "negative timeout",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, Timeout: -time.Second},
			wantErr: "timeout must not be negative",
		},
		{
			name:    "negative root keys ttl",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, RootKeysTTL: -time.Second},
			wantErr: "root_keys_ttl must not be negative",
		},
		{
			name:    "negative status cache ttl",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, StatusCacheTTL: -time.Second},
			wantErr: "status_cache_ttl must not be negative",
		},
		{
			name:    "negative clock skew",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, ClockSkew: -time.Second},
			wantErr: "clock_skew must not be negative",
		},
		{
			name:    "clock skew above what the verifier honours",
			config:  Config{TrustedLogHosts: []string{testLogHost}, RootKeys: []string{testRootKeyLine}, ClockSkew: scitt.MaxClockSkew + time.Second},
			wantErr: "clock_skew must not exceed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := tt.config
			before.TrustedLogHosts = slices.Clone(tt.config.TrustedLogHosts)
			before.RootKeys = slices.Clone(tt.config.RootKeys)

			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, before, tt.config, "Validate must not change the configuration")
		})
	}
}

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "plain host is lowercased", input: "Log.Example.COM", want: "log.example.com"},
		{name: "default https port is dropped", input: "log.example.com:443", want: "log.example.com"},
		{name: "other port is kept", input: "localhost:18443", want: "localhost:18443"},
		{name: "surrounding whitespace is trimmed", input: "  log.example.com ", want: "log.example.com"},
		{name: "ipv6 literal keeps brackets", input: "[::1]:18443", want: "[::1]:18443"},
		{name: "ipv6 literal on default port", input: "[::1]:443", want: "[::1]"},
		{name: "bare ipv6 literal", input: "[::1]", want: "[::1]"},
		{name: "bare ipv6 literal is lowercased", input: "[FE80::1]", want: "[fe80::1]"},
		{name: "empty", input: "   ", wantErr: "must not be empty"},
		{name: "port only", input: ":443", wantErr: "has no host name"},
		{name: "colon only", input: ":", wantErr: "has no host name"},
		{name: "empty brackets with port", input: "[]:443", wantErr: "has no host name"},
		{name: "empty brackets", input: "[]", wantErr: "has no host name"},
		{name: "scheme is rejected", input: "https://log.example.com", wantErr: "must not contain a scheme or path"},
		{name: "path is rejected", input: "log.example.com/v1", wantErr: "must not contain a scheme or path"},
		{name: "malformed port", input: "log.example.com:443:1", wantErr: "must be host or host:port"},
		{name: "unbracketed ipv6 literal", input: "::1", wantErr: "must be host or host:port"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeHost(tt.input)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeHostsDropsDuplicates(t *testing.T) {
	hosts, err := normalizeHosts([]string{"Log.Example.com:443", "log.example.com", "other.example.com"})
	require.NoError(t, err)
	assert.Equal(t, []string{"log.example.com", "other.example.com"}, hosts)
}
