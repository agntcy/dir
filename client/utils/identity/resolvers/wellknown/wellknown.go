// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknownresolver

import (
	"context"
	"crypto"
	"fmt"
	"strings"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/identity/resolvers/internal/keyutil"
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
	rest, ok := strings.CutPrefix(subject, "https://")
	if !ok {
		return nil, fmt.Errorf("invalid https subject %q", subject)
	}

	hostPart, _, _ := strings.Cut(rest, "/")

	u, err := keyutil.ParseHost(hostPart)
	if err != nil {
		return nil, fmt.Errorf("invalid https subject %q: %w", subject, err)
	}

	docURL := "https://" + u.Host + WellKnownPath

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

		if pub, ok := keyutil.PublicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}

	found, err := keyutil.Finish(keys, docURL)
	if err != nil {
		return nil, fmt.Errorf("wellknown: %w", err)
	}

	return found, nil
}
