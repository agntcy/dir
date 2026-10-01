// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknownresolver

import (
	"context"
	"crypto"
	"fmt"
	"net/url"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// WellKnownPath is where a domain publishes its JWKS (RFC 7517).
const WellKnownPath = "/.well-known/jwks.json"

// Resolver resolves "https://" subjects from the domain's JWKS file.
type Resolver struct {
	fetch resolvers.Fetcher
}

// New creates a Resolver. A nil fetch uses a default safefetch.Client.
func New(fetch resolvers.Fetcher) *Resolver {
	if fetch == nil {
		fetch = safefetch.New()
	}

	return &Resolver{fetch: fetch}
}

// Resolve implements resolvers.Resolver. It ignores certificate.
func (w *Resolver) Resolve(ctx context.Context, subject string, _ []byte) ([]crypto.PublicKey, error) {
	u, err := url.Parse(subject)
	// User must be refused: in "https://acme.com@evil.com" the host is evil.com.
	if err != nil || u.Scheme != "https" || u.User != nil || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid https subject %q", subject)
	}

	docURL := (&url.URL{Scheme: "https", Host: u.Host, Path: WellKnownPath}).String()

	body, err := w.fetch.Get(ctx, docURL)
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS from %s: %w", docURL, err)
	}

	set, err := jwk.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse JWKS from %s: %w", docURL, err)
	}

	keys := make([]crypto.PublicKey, 0, set.Len())

	for i := range set.Len() {
		key, ok := set.Key(i)
		if !ok {
			continue
		}

		if pub, ok := jws.PublicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("%w at %s", resolvers.ErrNoKeys, docURL)
	}

	return keys, nil
}
