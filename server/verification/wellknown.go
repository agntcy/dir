// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package verification

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/agntcy/dir/utils/logging"
)

const (
	// defaultMaxBodySize is the maximum size of the response body (1MB).
	defaultMaxBodySize = 1024 * 1024
)

var wellKnownLogger = logging.Logger("verification/wellknown")

// WellKnownFetcher handles fetching the well-known OASF file from domains.
type WellKnownFetcher struct {
	// client is the HTTP client to use for requests.
	client *http.Client

	// timeout is the maximum time to wait for HTTP requests.
	timeout time.Duration

	// maxBodySize is the maximum size of the response body to read.
	maxBodySize int64

	// allowInsecure allows HTTP instead of HTTPS (for testing only).
	allowInsecure bool
}

// WellKnownFetcherOption configures a WellKnownFetcher.
type WellKnownFetcherOption func(*WellKnownFetcher)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) WellKnownFetcherOption {
	return func(f *WellKnownFetcher) {
		f.client = client
	}
}

// WithHTTPTimeout sets the HTTP request timeout.
func WithHTTPTimeout(timeout time.Duration) WellKnownFetcherOption {
	return func(f *WellKnownFetcher) {
		f.timeout = timeout
	}
}

// WithMaxBodySize sets the maximum response body size.
func WithMaxBodySize(size int64) WellKnownFetcherOption {
	return func(f *WellKnownFetcher) {
		f.maxBodySize = size
	}
}

// WithAllowInsecure allows HTTP instead of HTTPS for well-known file fetching.
// WARNING: Only use for local development/testing. Never enable in production.
func WithAllowInsecure(allow bool) WellKnownFetcherOption {
	return func(f *WellKnownFetcher) {
		f.allowInsecure = allow
	}
}

// NewWellKnownFetcher creates a new well-known file fetcher with the given options.
func NewWellKnownFetcher(opts ...WellKnownFetcherOption) *WellKnownFetcher {
	f := &WellKnownFetcher{
		timeout:     DefaultHTTPTimeout,
		maxBodySize: defaultMaxBodySize,
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

// FetchKeys retrieves public keys from the well-known OASF file for the given domain.
// It fetches https://<domain>/.well-known/oasf.json and parses the keys.
func (f *WellKnownFetcher) FetchKeys(ctx context.Context, domain string) ([]PublicKey, error) {
	// Create context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Build the URL
	scheme := "https"
	if f.allowInsecure {
		scheme = "http"
	}

	url := scheme + "://" + domain + WellKnownPath

	wellKnownLogger.Debug("Fetching well-known OASF file", "domain", domain, "url", url)

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
		wellKnownLogger.Debug("Well-known file not found", "domain", domain)

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

	wellKnownLogger.Debug("Received well-known file", "domain", domain, "size", len(body))

	// Parse JSON
	var wellKnown WellKnownFile
	if err := json.Unmarshal(body, &wellKnown); err != nil {
		return nil, fmt.Errorf("failed to parse well-known file: %w", err)
	}

	// Validate version
	if wellKnown.Version != 1 {
		return nil, fmt.Errorf("unsupported well-known file version: %d", wellKnown.Version)
	}

	// Parse keys
	keys := make([]PublicKey, 0, len(wellKnown.Keys))

	for _, wk := range wellKnown.Keys {
		key, err := ParseWellKnownKey(wk)
		if err != nil {
			wellKnownLogger.Warn("Failed to parse key from well-known file",
				"domain", domain, "keyID", wk.ID, "error", err)

			continue
		}

		keys = append(keys, *key)
		wellKnownLogger.Debug("Parsed public key from well-known file",
			"domain", domain, "keyID", key.ID, "keyType", key.Type)
	}

	return keys, nil
}
