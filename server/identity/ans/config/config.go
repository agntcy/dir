// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package config holds the configuration of the ans identity scheme: which
// transparency logs are trusted, how they are authenticated, and how long one
// verification may take.
package config

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"
)

const (
	// DefaultTimeout is the default total time budget for one ans verification
	// (DNS, then the transparency log fetches).
	DefaultTimeout = 10 * time.Second

	// httpsDefaultPort is dropped from normalized hosts so "log.example.com"
	// and "log.example.com:443" name the same origin.
	httpsDefaultPort = "443"
)

// Config configures the ans identity scheme.
type Config struct {
	// Enabled turns verification of ans:// subjects on.
	Enabled bool `json:"enabled,omitempty" mapstructure:"enabled"`

	// TrustedLogHosts lists the transparency-log hosts (host or host:port)
	// that badge records may point at. Any other host is rejected before a
	// request is made. Required when Enabled.
	TrustedLogHosts []string `json:"trusted_log_hosts,omitempty" mapstructure:"trusted_log_hosts"`

	// RootKeys pins the logs' signing keys as root-key lines
	// ("origin+kid+base64"), shared by all trusted logs. Required when Enabled
	// unless AllowUnpinnedRootKeys is set.
	RootKeys []string `json:"root_keys,omitempty" mapstructure:"root_keys"`

	// AllowUnpinnedRootKeys permits running without RootKeys. The keys are then
	// fetched from each log, so trust rests on TLS to the trusted hosts.
	AllowUnpinnedRootKeys bool `json:"allow_unpinned_root_keys,omitempty" mapstructure:"allow_unpinned_root_keys"`

	// Timeout is the total time budget for one ans verification. Defaults to
	// DefaultTimeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// DNSServer optionally names the resolver (host:port) for the _ans-badge
	// lookups instead of the system resolver.
	DNSServer string `json:"dns_server,omitempty" mapstructure:"dns_server"`

	// CAFile optionally adds PEM certificates to the system roots trusted for
	// transparency-log connections.
	CAFile string `json:"ca_file,omitempty" mapstructure:"ca_file"`
}

// Keys lists the configuration keys of Config in field order, as they appear
// under identity.ans in a configuration file. The environment binders of both
// binaries read this list, so a key added to Config is bound everywhere.
func Keys() []string {
	typ := reflect.TypeFor[Config]()
	keys := make([]string, 0, typ.NumField())

	for field := range typ.Fields() {
		keys = append(keys, field.Tag.Get("mapstructure"))
	}

	return keys
}

// GetTimeout returns the timeout with default fallback.
func (c *Config) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultTimeout
	}

	return c.Timeout
}

// Validate checks the configuration's syntax and normalizes TrustedLogHosts
// and RootKeys in place (trimmed, hosts lowercased with the default https
// port removed, empty entries dropped). It opens no file and parses no key;
// the verifier constructor does both. It reports nothing when Enabled is
// false.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	hosts, err := normalizeHosts(c.TrustedLogHosts)
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		return errors.New("ans: trusted_log_hosts must list at least one transparency-log host")
	}

	c.TrustedLogHosts = hosts

	c.RootKeys = trimmed(c.RootKeys)
	if len(c.RootKeys) == 0 && !c.AllowUnpinnedRootKeys {
		return errors.New("ans: root_keys is empty; pin the logs' root-key lines or set allow_unpinned_root_keys to trust the logs over TLS alone")
	}

	if c.Timeout < 0 {
		return fmt.Errorf("ans: timeout must not be negative, got %s", c.Timeout)
	}

	if c.DNSServer != "" {
		if _, _, err := net.SplitHostPort(c.DNSServer); err != nil {
			return fmt.Errorf("ans: dns_server must be host:port: %w", err)
		}
	}

	return nil
}

// NormalizeHost lowercases a host and drops an explicit default https port so
// configured hosts and hosts taken from badge URLs compare byte for byte.
// The input is "host", "host:port", "[ipv6]" or "[ipv6]:port"; IPv6 literals
// keep their brackets.
func NormalizeHost(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("ans: host must not be empty")
	}

	if strings.Contains(host, "/") {
		return "", fmt.Errorf("ans: host %q must not contain a scheme or path", host)
	}

	name, port, err := splitHostPort(host)
	if err != nil {
		return "", err
	}

	if name == "" {
		return "", fmt.Errorf("ans: host %q has no host name", host)
	}

	name = strings.ToLower(name)

	if port == "" || port == httpsDefaultPort {
		if strings.Contains(name, ":") {
			return "[" + name + "]", nil
		}

		return name, nil
	}

	return net.JoinHostPort(name, port), nil
}

// splitHostPort separates the optional port from host, accepting a bracketed
// IPv6 literal without one.
func splitHostPort(host string) (string, string, error) {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return host[1 : len(host)-1], "", nil
	}

	if !strings.Contains(host, ":") {
		return host, "", nil
	}

	name, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", "", fmt.Errorf("ans: host %q must be host or host:port: %w", host, err)
	}

	return name, port, nil
}

func normalizeHosts(hosts []string) ([]string, error) {
	seen := make(map[string]struct{}, len(hosts))
	out := make([]string, 0, len(hosts))

	for _, host := range hosts {
		if strings.TrimSpace(host) == "" {
			return nil, errors.New("ans: trusted_log_hosts contains an empty entry")
		}

		normalized, err := NormalizeHost(host)
		if err != nil {
			return nil, err
		}

		if _, dup := seen[normalized]; dup {
			continue
		}

		seen[normalized] = struct{}{}

		out = append(out, normalized)
	}

	return out, nil
}

func trimmed(values []string) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}

	return out
}
