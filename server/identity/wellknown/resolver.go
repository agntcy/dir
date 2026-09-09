// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package wellknown verifies "https://" subjects against the domain's JWKS
// well-known file (RFC 7517), producing identity/ownership claim
// verification results.
package wellknown

import (
	"context"
	"fmt"
	"strings"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// WellKnownPath is the path of the JWKS well-known file (RFC 7517).
const WellKnownPath = "/.well-known/jwks.json"

// Resolver verifies claim signatures against a domain's published JWKS.
type Resolver struct {
	fetch *safefetch.Client
}

// New creates a new https/JWKS resolver.
func New(fetch *safefetch.Client) *Resolver {
	if fetch == nil {
		fetch = safefetch.New()
	}

	return &Resolver{fetch: fetch}
}

// Scheme implements identity.Resolver.
func (*Resolver) Scheme() string {
	return "https"
}

// Verify implements identity.Resolver.
func (r *Resolver) Verify(ctx context.Context, subject string, signature, payload []byte) (bool, error) {
	domain, err := domainFromSubject(subject)
	if err != nil {
		return false, err
	}

	url := "https://" + domain + WellKnownPath

	keySet, err := jwk.Fetch(ctx, url, jwk.WithHTTPClient(r.fetch.HTTPClient()))
	if err != nil {
		return false, fmt.Errorf("fetch JWKS from %s: %w", url, err)
	}

	return verifyAgainstKeySet(keySet, signature, payload), nil
}

// verifyAgainstKeySet tries signature over payload against every key in
// keySet. Split out from Verify so it can be unit-tested without a network
// fetch.
func verifyAgainstKeySet(keySet jwk.Set, signature, payload []byte) bool {
	for i := range keySet.Len() {
		key, ok := keySet.Key(i)
		if !ok {
			continue
		}

		var rawKey any
		if err := key.Raw(&rawKey); err != nil {
			continue
		}

		if identityv1.VerifyJWS(string(signature), rawKey, payload) == nil {
			return true
		}
	}

	return false
}

// domainFromSubject extracts the domain from an "https://domain[/path]" (or
// bare "domain") subject.
func domainFromSubject(subject string) (string, error) {
	domain := strings.TrimPrefix(subject, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	if idx := strings.IndexByte(domain, '/'); idx >= 0 {
		domain = domain[:idx]
	}

	if domain == "" {
		return "", fmt.Errorf("invalid https subject: %q", subject)
	}

	return domain, nil
}
