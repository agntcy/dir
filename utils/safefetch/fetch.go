// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package safefetch provides an SSRF-safe HTTP GET client for fetching
// documents (JWKS files, DID documents) from URLs derived from untrusted,
// attacker-influenceable input.
package safefetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"time"
)

const (
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 10 * time.Second

	// DefaultMaxBytes is the default response size cap.
	DefaultMaxBytes = 1 << 20 // 1 MiB

	dialTimeout      = 5 * time.Second
	handshakeTimeout = 5 * time.Second
	maxRedirects     = 3
)

// ErrDisallowedAddress is returned when a target host resolves to a
// non-public address (private, loopback, link-local, and so on).
var ErrDisallowedAddress = errors.New("target address is not allowed")

// reservedPrefixes are non-public ranges that netip's own predicates don't
// cover. 100.64.0.0/10 hosts some clouds' metadata endpoints, and the IPv6
// transition ranges (NAT64, 6to4, Teredo) can embed or translate to a blocked
// IPv4 target, e.g. 64:ff9b::a9fe:a9fe reaches 169.254.169.254.
var reservedPrefixes = []netip.Prefix{
	// IPv4.
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	// IPv6.
	netip.MustParsePrefix("::/96"),          // deprecated IPv4-compatible
	netip.MustParsePrefix("64:ff9b::/96"),   // NAT64
	netip.MustParsePrefix("64:ff9b:1::/48"), // local-use NAT64
	netip.MustParsePrefix("100::/64"),       // discard-only
	netip.MustParsePrefix("2001::/32"),      // Teredo
	netip.MustParsePrefix("2001:db8::/32"),  // documentation
	netip.MustParsePrefix("2002::/16"),      // 6to4
	netip.MustParsePrefix("fec0::/10"),      // deprecated site-local
}

// Client performs SSRF-safe HTTP GET requests: https only unless
// WithAllowHTTP is set, no connections to non-public addresses (checked at
// dial time, after DNS resolution, so DNS rebinding can't bypass it), a
// bounded redirect chain that keeps the scheme rule, and a response size cap.
type Client struct {
	httpClient *http.Client
	maxBytes   int64
	allowHTTP  bool
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the overall request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = timeout }
}

// WithMaxBytes sets the maximum response size accepted.
func WithMaxBytes(maxBytes int64) Option {
	return func(c *Client) { c.maxBytes = maxBytes }
}

// WithAllowHTTP permits plain http:// requests (for local testing only).
func WithAllowHTTP() Option {
	return func(c *Client) { c.allowHTTP = true }
}

// New creates a Client that refuses to connect to non-public addresses.
func New(opts ...Option) *Client {
	return newClient(isDisallowedAddr, opts...)
}

func newClient(blocked func(netip.Addr) bool, opts ...Option) *Client {
	c := &Client{maxBytes: DefaultMaxBytes}

	c.httpClient = &http.Client{
		Timeout:       DefaultTimeout,
		CheckRedirect: c.checkRedirect,
		Transport: &http.Transport{
			DialContext:         safeDialContext(&net.Dialer{Timeout: dialTimeout}, blocked),
			TLSHandshakeTimeout: handshakeTimeout,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Get fetches rawURL and returns its body, up to the configured size cap.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if err := c.checkScheme(req.URL); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %d", rawURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, c.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if int64(len(body)) > c.maxBytes {
		return nil, fmt.Errorf("response exceeds maximum size of %d bytes", c.maxBytes)
	}

	return body, nil
}

func (c *Client) checkScheme(u *url.URL) error {
	if u.Scheme == "https" || (c.allowHTTP && u.Scheme == "http") {
		return nil
	}

	return fmt.Errorf("scheme %q is not allowed", u.Scheme)
}

func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("too many redirects")
	}

	return c.checkScheme(req.URL)
}

// safeDialContext resolves the target host, rejects blocked addresses, and
// connects to the validated IP directly rather than re-resolving the name,
// which closes the window between validation and connection.
func safeDialContext(dialer *net.Dialer, blocked func(netip.Addr) bool) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("split host/port %q: %w", addr, err)
		}

		ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", host, err)
		}

		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses found for %q", host)
		}

		for _, ip := range ips {
			if blocked(ip) {
				return nil, fmt.Errorf("%w: %s resolves to %s", ErrDisallowedAddress, host, ip)
			}
		}

		return dialFirstReachable(ctx, dialer, network, ips, port)
	}
}

// dialFirstReachable tries the already-validated addresses in order, so a
// down endpoint of a multi-address host doesn't make the whole host unreachable.
func dialFirstReachable(ctx context.Context, dialer *net.Dialer, network string, ips []netip.Addr, port string) (net.Conn, error) {
	var errs []error

	for _, ip := range ips {
		conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.Unmap().String(), port))
		if err == nil {
			return conn, nil
		}

		errs = append(errs, err)
	}

	return nil, fmt.Errorf("dial: %w", errors.Join(errs...))
}

// isDisallowedAddr reports whether addr is anything other than a public
// unicast address.
func isDisallowedAddr(addr netip.Addr) bool {
	addr = addr.Unmap()

	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return true
	}

	for _, prefix := range reservedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}
