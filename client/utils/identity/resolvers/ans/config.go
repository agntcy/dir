// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package ansresolver

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/agentnameservice/ans-sdk-go/verify/scitt"
)

const (
	// DefaultTimeout is the time allowed for each of the two network stages of
	// one lookup: the publisher's DNS record, then the transparency log.
	DefaultTimeout = 10 * time.Second

	// DefaultRootKeysTTL is how long root keys fetched from a transparency log
	// are used before they are fetched again (unpinned mode only).
	DefaultRootKeysTTL = 10 * time.Minute

	// DefaultStatusCacheTTL is how long a subject's attestation (its badge
	// record and the log's answer) is reused across claims. It is the most a
	// revocation can be delayed on top of the verification interval.
	DefaultStatusCacheTTL = 30 * time.Second

	// DefaultClockSkew is the tolerance applied to a status token's expiry.
	DefaultClockSkew = 30 * time.Second

	// httpsDefaultPort is dropped from normalized hosts so "log.example.com"
	// and "log.example.com:443" name the same origin.
	httpsDefaultPort = "443"
)

// Config configures the ans:// resolver. New validates it; Validate reads it
// and changes nothing.
type Config struct {
	// TrustedLogHosts lists the transparency-log hosts (host or host:port) a
	// badge record may point at. Any other host is refused before a request
	// is made.
	TrustedLogHosts []string `json:"trusted_log_hosts,omitempty" mapstructure:"trusted_log_hosts"`

	// RootKeys pins the logs' signing keys as the lines their /root-keys
	// endpoint serves ("name+kid+base64"), shared by every trusted log.
	// Required unless AllowUnpinnedRootKeys is set.
	RootKeys []string `json:"root_keys,omitempty" mapstructure:"root_keys"`

	// AllowUnpinnedRootKeys fetches each log's keys from its /root-keys
	// endpoint instead, so trust rests on TLS to the trusted hosts. It cannot
	// be combined with RootKeys.
	AllowUnpinnedRootKeys bool `json:"allow_unpinned_root_keys,omitempty" mapstructure:"allow_unpinned_root_keys"`

	// RootKeysTTL is how long fetched root keys are used before they are
	// fetched again. Defaults to DefaultRootKeysTTL.
	RootKeysTTL time.Duration `json:"root_keys_ttl,omitempty" mapstructure:"root_keys_ttl"`

	// StatusCacheTTL is how long a subject's attestation is reused across
	// claims. Defaults to DefaultStatusCacheTTL; keep it below the
	// verification interval.
	StatusCacheTTL time.Duration `json:"status_cache_ttl,omitempty" mapstructure:"status_cache_ttl"`

	// Timeout is the time allowed for each network stage of one lookup.
	// Defaults to DefaultTimeout.
	Timeout time.Duration `json:"timeout,omitempty" mapstructure:"timeout"`

	// ClockSkew is the tolerance applied to a status token's expiry. Defaults
	// to DefaultClockSkew; at most scitt.MaxClockSkew.
	ClockSkew time.Duration `json:"clock_skew,omitempty" mapstructure:"clock_skew"`
}

// GetTimeout returns the timeout with default fallback.
func (c Config) GetTimeout() time.Duration {
	if c.Timeout == 0 {
		return DefaultTimeout
	}

	return c.Timeout
}

// GetRootKeysTTL returns the root-keys lifetime with default fallback.
func (c Config) GetRootKeysTTL() time.Duration {
	if c.RootKeysTTL == 0 {
		return DefaultRootKeysTTL
	}

	return c.RootKeysTTL
}

// GetStatusCacheTTL returns the attestation lifetime with default fallback.
func (c Config) GetStatusCacheTTL() time.Duration {
	if c.StatusCacheTTL == 0 {
		return DefaultStatusCacheTTL
	}

	return c.StatusCacheTTL
}

// GetClockSkew returns the clock-skew tolerance with default fallback.
func (c Config) GetClockSkew() time.Duration {
	if c.ClockSkew == 0 {
		return DefaultClockSkew
	}

	return c.ClockSkew
}

// Validate reports the first problem with the configuration: no trusted log
// host or a malformed one, neither pinned keys nor the unpinned opt-in (or
// both), a negative duration, or a clock skew above what the token verifier
// honours. It opens no file and parses no key; New does that.
func (c Config) Validate() error {
	if _, err := c.trustedSet(); err != nil {
		return err
	}

	return c.validateTrust()
}

// trustedSet returns the normalized trusted log hosts as a set.
func (c Config) trustedSet() (map[string]struct{}, error) {
	hosts, err := normalizeHosts(c.TrustedLogHosts)
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, errors.New("ans: trusted_log_hosts must list at least one transparency-log host")
	}

	trusted := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		trusted[host] = struct{}{}
	}

	return trusted, nil
}

// validateTrust checks the key policy and the tunables.
func (c Config) validateTrust() error {
	pinned := len(trimmed(c.RootKeys)) > 0

	switch {
	case pinned && c.AllowUnpinnedRootKeys:
		return errors.New("ans: root_keys and allow_unpinned_root_keys cannot both be set")
	case !pinned && !c.AllowUnpinnedRootKeys:
		return errors.New("ans: root_keys is empty; pin the logs' root-key lines or set allow_unpinned_root_keys to trust the logs over TLS alone")
	}

	for _, d := range []struct {
		name  string
		value time.Duration
	}{
		{"root_keys_ttl", c.RootKeysTTL},
		{"status_cache_ttl", c.StatusCacheTTL},
		{"timeout", c.Timeout},
		{"clock_skew", c.ClockSkew},
	} {
		if d.value < 0 {
			return fmt.Errorf("ans: %s must not be negative, got %s", d.name, d.value)
		}
	}

	if c.ClockSkew > scitt.MaxClockSkew {
		return fmt.Errorf("ans: clock_skew must not exceed %s, got %s", scitt.MaxClockSkew, c.ClockSkew)
	}

	return nil
}

// normalizeHost lowercases a host and drops an explicit default https port
// so configured hosts and hosts taken from badge URLs compare byte for byte.
// The input is "host", "host:port", "[ipv6]" or "[ipv6]:port"; IPv6 literals
// keep their brackets.
func normalizeHost(host string) (string, error) {
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

// normalizeHosts normalizes every host and drops duplicates, keeping order.
func normalizeHosts(hosts []string) ([]string, error) {
	seen := make(map[string]struct{}, len(hosts))
	out := make([]string, 0, len(hosts))

	for _, host := range hosts {
		if strings.TrimSpace(host) == "" {
			return nil, errors.New("ans: trusted_log_hosts contains an empty entry")
		}

		normalized, err := normalizeHost(host)
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

// trimmed returns the non-empty values, each without surrounding whitespace.
func trimmed(values []string) []string {
	out := make([]string, 0, len(values))

	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}

	return out
}
