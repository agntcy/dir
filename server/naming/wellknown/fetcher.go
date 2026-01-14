// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package wellknown provides well-known file verification for name ownership.
package wellknown

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/agntcy/dir/server/naming"
	"github.com/agntcy/dir/server/naming/wellknown/config"
	"github.com/agntcy/dir/utils/logging"
)

// WellKnownPath is the path for the OASF well-known file.
const WellKnownPath = "/.well-known/oasf.json"

var logger = logging.Logger("naming/wellknown")

// Fetcher handles fetching the well-known OASF file from domains.
type Fetcher struct {
	// client is the HTTP client to use for requests.
	client *http.Client

	// timeout is the maximum time to wait for HTTP requests.
	timeout time.Duration

	// maxBodySize is the maximum size of the response body to read.
	maxBodySize int64

	// allowInsecure allows HTTP instead of HTTPS (for testing only).
	allowInsecure bool
}

// Option configures a Fetcher.
type Option func(*Fetcher)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(f *Fetcher) {
		f.client = client
	}
}

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(f *Fetcher) {
		f.timeout = timeout
	}
}

// WithMaxBodySize sets the maximum response body size.
func WithMaxBodySize(size int64) Option {
	return func(f *Fetcher) {
		f.maxBodySize = size
	}
}

// WithAllowInsecure allows HTTP instead of HTTPS for well-known file fetching.
// WARNING: Only use for local development/testing. Never enable in production.
func WithAllowInsecure(allow bool) Option {
	return func(f *Fetcher) {
		f.allowInsecure = allow
	}
}

// NewFetcher creates a new well-known file fetcher with the given options.
func NewFetcher(opts ...Option) *Fetcher {
	f := &Fetcher{
		timeout:     config.DefaultTimeout,
		maxBodySize: config.DefaultMaxBodySize,
	}

	for _, opt := range opts {
		opt(f)
	}

	// Create default client if not provided
	if f.client == nil {
		f.client = &http.Client{
			Timeout: f.timeout,
		}
	}

	return f
}

// NewFetcherFromConfig creates a new well-known fetcher from configuration.
func NewFetcherFromConfig(cfg *config.Config) *Fetcher {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	return NewFetcher(
		WithTimeout(cfg.Timeout),
		WithMaxBodySize(cfg.MaxBodySize),
		WithAllowInsecure(cfg.AllowInsecure),
	)
}

// LookupKeys retrieves public keys from the well-known OASF file for the given domain.
// It fetches https://<domain>/.well-known/oasf.json and parses the keys.
// This method implements the naming.KeyLookup interface.
func (f *Fetcher) LookupKeys(ctx context.Context, domain string) ([]naming.PublicKey, error) {
	// Create context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Build the URL
	scheme := "https"
	if f.allowInsecure {
		scheme = "http"
	}

	url := scheme + "://" + domain + WellKnownPath

	logger.Debug("Fetching well-known OASF file", "domain", domain, "url", url)

	// Create request
	req, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "AGNTCY-Directory/1.0")

	// Make request
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode == http.StatusNotFound {
		logger.Debug("Well-known file not found", "domain", domain)

		return nil, nil // Not an error, just no file
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
	}

	// Read body with size limit
	limitedReader := io.LimitReader(resp.Body, f.maxBodySize)

	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	logger.Debug("Received well-known file", "domain", domain, "size", len(body))

	// Parse JSON
	var wellKnown naming.WellKnownFile
	if err := json.Unmarshal(body, &wellKnown); err != nil {
		return nil, fmt.Errorf("failed to parse well-known file: %w", err)
	}

	// Validate version
	if wellKnown.Version != 1 {
		return nil, fmt.Errorf("unsupported well-known file version: %d", wellKnown.Version)
	}

	// Parse keys
	keys := make([]naming.PublicKey, 0, len(wellKnown.Keys))

	for _, wk := range wellKnown.Keys {
		key, err := ParseKey(wk)
		if err != nil {
			logger.Warn("Failed to parse key from well-known file",
				"domain", domain, "keyID", wk.ID, "error", err)

			continue
		}

		keys = append(keys, *key)
		logger.Debug("Parsed public key from well-known file",
			"domain", domain, "keyID", key.ID, "keyType", key.Type)
	}

	return keys, nil
}
