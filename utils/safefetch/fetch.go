// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package safefetch provides an SSRF-safe HTTP GET client for use by
// identity resolvers that fetch remote documents (JWKS well-known files, DID
// documents) referenced by untrusted, attacker-influenceable URIs.
package safefetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 10 * time.Second

	// DefaultMaxBytes is the default response size cap.
	DefaultMaxBytes = 1 << 20 // 1 MiB

	dialTimeout = 5 * time.Second
)

// ErrDisallowedAddress is returned when a target host resolves to a
// private/loopback/link-local/cloud-metadata address.
var ErrDisallowedAddress = errors.New("target address is not allowed")

// Client performs SSRF-safe HTTP GET requests: https-only by default, blocks
// requests to private/loopback/link-local/multicast/cloud-metadata
// destinations (re-checked at dial time, after DNS resolution, to prevent
// DNS-rebinding), and enforces a response size cap.
type Client struct {
	httpClient *http.Client
	maxBytes   int64
	allowHTTP  bool
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithMaxBytes sets the maximum response size accepted.
func WithMaxBytes(maxBytes int64) Option {
	return func(c *Client) {
		c.maxBytes = maxBytes
	}
}

// WithAllowHTTP permits plain http:// requests (for local testing only).
func WithAllowHTTP() Option {
	return func(c *Client) {
		c.allowHTTP = true
	}
}

// New creates a new SSRF-safe HTTP client.
func New(opts ...Option) *Client {
	c := &Client{
		maxBytes: DefaultMaxBytes,
	}

	dialer := &net.Dialer{Timeout: dialTimeout}
	c.httpClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			DialContext: safeDialContext(dialer),
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// HTTPClient returns the underlying *http.Client, for libraries (e.g. jwx)
// that accept one directly.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// Get fetches rawURL and returns its body, up to the configured size cap.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if req.URL.Scheme != "https" && !(c.allowHTTP && req.URL.Scheme == "http") {
		return nil, fmt.Errorf("scheme %q is not allowed", req.URL.Scheme)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %d", rawURL, resp.StatusCode)
	}

	return readLimited(resp.Body, c.maxBytes)
}

// readLimited reads at most maxBytes+1 bytes from r, returning an error if
// the response is larger than maxBytes.
func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("response exceeds maximum size of %d bytes", maxBytes)
	}

	return body, nil
}

// safeDialContext returns a DialContext that resolves the target host,
// rejects disallowed addresses, and connects to the validated IP directly
// (rather than re-resolving via the original hostname) to close the
// DNS-rebinding TOCTOU window between validation and connection.
func safeDialContext(dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("split host/port %q: %w", addr, err)
		}

		if ip := net.ParseIP(host); ip != nil {
			if isDisallowedIP(ip) {
				return nil, fmt.Errorf("%w: %s", ErrDisallowedAddress, ip)
			}

			return dialer.DialContext(ctx, network, addr)
		}

		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", host, err)
		}

		for _, ip := range ips {
			if isDisallowedIP(ip) {
				return nil, fmt.Errorf("%w: %s resolves to %s", ErrDisallowedAddress, host, ip)
			}
		}

		if len(ips) == 0 {
			return nil, fmt.Errorf("no addresses found for %q", host)
		}

		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
}

// isDisallowedIP reports whether ip is a private, loopback, link-local,
// multicast, or unspecified address (this also covers the common cloud
// metadata endpoint 169.254.169.254, which falls under link-local).
func isDisallowedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsPrivate() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}
