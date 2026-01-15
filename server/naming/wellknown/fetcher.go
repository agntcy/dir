// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package wellknown provides well-known file verification for name ownership
// using the JWKS (JSON Web Key Set) standard (RFC 7517).
package wellknown

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/agntcy/dir/server/naming"
	"github.com/agntcy/dir/server/naming/wellknown/config"
	"github.com/agntcy/dir/utils/logging"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// WellKnownPath is the path for the JWKS well-known file (RFC 7517).
const WellKnownPath = "/.well-known/jwks.json"

var logger = logging.Logger("naming/wellknown")

// Fetcher handles fetching the JWKS well-known file from domains.
type Fetcher struct {
	// client is the HTTP client to use for requests.
	client *http.Client

	// timeout is the maximum time to wait for HTTP requests.
	timeout time.Duration
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

// NewFetcher creates a new well-known file fetcher with the given options.
func NewFetcher(opts ...Option) *Fetcher {
	f := &Fetcher{
		timeout: config.DefaultTimeout,
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
	)
}

// LookupKeysWithScheme retrieves public keys from the JWKS well-known file for the given domain.
// The scheme parameter specifies whether to use "http" or "https".
// This method implements the naming.KeyLookupWithScheme interface.
func (f *Fetcher) LookupKeysWithScheme(ctx context.Context, domain, scheme string) ([]naming.PublicKey, error) {
	// Create context with timeout
	fetchCtx, cancel := context.WithTimeout(ctx, f.timeout)
	defer cancel()

	// Build the URL
	url := scheme + "://" + domain + WellKnownPath

	logger.Debug("Fetching JWKS well-known file", "domain", domain, "url", url)

	// Fetch and parse JWKS using the jwx library
	keySet, err := jwk.Fetch(fetchCtx, url, jwk.WithHTTPClient(f.client))
	if err != nil {
		logger.Debug("Failed to fetch JWKS", "domain", domain, "error", err)

		return nil, fmt.Errorf("failed to fetch JWKS from %s: %w", url, err)
	}

	logger.Debug("Received JWKS file", "domain", domain, "keyCount", keySet.Len())

	// Convert JWK keys to our PublicKey format
	keys := make([]naming.PublicKey, 0, keySet.Len())

	for iter := keySet.Keys(fetchCtx); iter.Next(fetchCtx); {
		key, ok := iter.Pair().Value.(jwk.Key)
		if !ok {
			logger.Warn("Failed to cast to jwk.Key", "domain", domain)

			continue
		}

		publicKey, err := ConvertJWKToPublicKey(key)
		if err != nil {
			logger.Warn("Failed to convert JWK to public key",
				"domain", domain, "kid", key.KeyID(), "error", err)

			continue
		}

		keys = append(keys, *publicKey)
		logger.Debug("Parsed public key from JWKS",
			"domain", domain, "keyID", publicKey.ID, "keyType", publicKey.Type)
	}

	return keys, nil
}
