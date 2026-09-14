// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestConfigGetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{name: "zero uses default", timeout: 0, want: DefaultTimeout},
		{name: "custom timeout", timeout: 3 * time.Second, want: 3 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{Timeout: tt.timeout}
			if got := c.GetTimeout(); got != tt.want {
				t.Errorf("GetTimeout() = %v, want %v", got, tt.want)
			}
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
		{name: "ipv6 literal on another port", input: "[::1]:8443", want: "[::1]:8443"},
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
			got, err := NormalizeHost(tt.input)

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("NormalizeHost(%q) error = %v, want containing %q", tt.input, err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("NormalizeHost(%q) unexpected error: %v", tt.input, err)
			}

			if got != tt.want {
				t.Errorf("NormalizeHost(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantErr   string
		wantHosts []string
		wantKeys  []string
	}{
		{
			name:   "disabled needs nothing",
			config: Config{},
		},
		{
			name: "enabled with hosts and pinned keys",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{" Log.Example.com:443 ", "localhost:18443", "log.example.com"},
				RootKeys:        []string{" ans-demo+1a2b3c4d+AjBZ ", ""},
			},
			wantHosts: []string{"log.example.com", "localhost:18443"},
			wantKeys:  []string{"ans-demo+1a2b3c4d+AjBZ"},
		},
		{
			name: "enabled without hosts",
			config: Config{
				Enabled:  true,
				RootKeys: []string{"ans-demo+1a2b3c4d+AjBZ"},
			},
			wantErr: "trusted_log_hosts must list at least one",
		},
		{
			name: "empty host entry",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"log.example.com", " "},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
			},
			wantErr: "contains an empty entry",
		},
		{
			name: "host with scheme",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"https://log.example.com"},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
			},
			wantErr: "must not contain a scheme or path",
		},
		{
			name: "host without a name",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{":443"},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
			},
			wantErr: "has no host name",
		},
		{
			name: "no root keys without opt-in",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"log.example.com"},
			},
			wantErr: "root_keys is empty",
		},
		{
			name: "no root keys with opt-in",
			config: Config{
				Enabled:               true,
				TrustedLogHosts:       []string{"log.example.com"},
				AllowUnpinnedRootKeys: true,
			},
			wantHosts: []string{"log.example.com"},
			wantKeys:  []string{},
		},
		{
			name: "negative timeout",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"log.example.com"},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
				Timeout:         -time.Second,
			},
			wantErr: "timeout must not be negative",
		},
		{
			name: "dns server without port",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"log.example.com"},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
				DNSServer:       "127.0.0.1",
			},
			wantErr: "dns_server must be host:port",
		},
		{
			name: "dns server with port",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"log.example.com"},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
				DNSServer:       "127.0.0.1:15353",
			},
			wantHosts: []string{"log.example.com"},
			wantKeys:  []string{"ans-demo+1a2b3c4d+AjBZ"},
		},
		{
			name: "ca file is not opened",
			config: Config{
				Enabled:         true,
				TrustedLogHosts: []string{"log.example.com"},
				RootKeys:        []string{"ans-demo+1a2b3c4d+AjBZ"},
				CAFile:          filepath.Join(t.TempDir(), "missing.pem"),
			},
			wantHosts: []string{"log.example.com"},
			wantKeys:  []string{"ans-demo+1a2b3c4d+AjBZ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.config

			err := cfg.Validate()

			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Validate() error = %v, want containing %q", err, tt.wantErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("Validate() unexpected error: %v", err)
			}

			if tt.wantHosts != nil && !slices.Equal(cfg.TrustedLogHosts, tt.wantHosts) {
				t.Errorf("TrustedLogHosts = %v, want %v", cfg.TrustedLogHosts, tt.wantHosts)
			}

			if tt.wantKeys != nil && !slices.Equal(cfg.RootKeys, tt.wantKeys) {
				t.Errorf("RootKeys = %v, want %v", cfg.RootKeys, tt.wantKeys)
			}
		})
	}
}

// Keys drives the environment binders of both binaries, so a field added to
// Config must show up here, and the author adding it must bind and document
// it.
func TestKeys(t *testing.T) {
	want := []string{"enabled", "trusted_log_hosts", "root_keys", "allow_unpinned_root_keys", "timeout", "dns_server", "ca_file"}

	if got := Keys(); !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}
