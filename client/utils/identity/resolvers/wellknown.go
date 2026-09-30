// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"context"
	"crypto"
	"fmt"
	"strings"

	"github.com/agntcy/dir/utils/safefetch"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// WellKnownPath is where a domain publishes its JWKS (RFC 7517).
const WellKnownPath = "/.well-known/jwks.json"

// WellKnown resolves "https://" subjects from the domain's JWKS file.
type WellKnown struct {
	fetch Fetcher
}

// NewWellKnown creates a JWKS resolver. A nil fetch uses a default
// safefetch.Client.
func NewWellKnown(fetch Fetcher) *WellKnown {
	if fetch == nil {
		fetch = safefetch.New()
	}

	return &WellKnown{fetch: fetch}
}

// Resolve implements Resolver. It ignores certificate.
func (w *WellKnown) Resolve(ctx context.Context, subject string, _ []byte) ([]crypto.PublicKey, error) {
	rest, ok := strings.CutPrefix(subject, "https://")
	if !ok {
		return nil, fmt.Errorf("invalid https subject %q", subject)
	}

	hostPart, _, _ := strings.Cut(rest, "/")

	u, err := parseHost(hostPart)
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

		if pub, ok := publicKeyFromJWK(key); ok {
			keys = append(keys, pub)
		}
	}

	return finish(keys, docURL)
}

// publicKeyFromJWK returns the signature-verification public key of key,
// skipping encryption keys and key types jws can't verify with.
func publicKeyFromJWK(key jwk.Key) (crypto.PublicKey, bool) {
	if key.KeyUsage() == string(jwk.ForEncryption) {
		return nil, false
	}

	pub, err := jwk.PublicKeyOf(key)
	if err != nil {
		return nil, false
	}

	var raw any
	if err := pub.Raw(&raw); err != nil {
		return nil, false
	}

	return toPublicKey(raw)
}
