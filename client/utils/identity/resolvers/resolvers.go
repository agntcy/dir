// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package resolvers finds the public keys a claim's subject publishes: a DNS
// TXT record, a DID document, a JWKS well-known file, or a SPIFFE X.509-SVID.
//
// A resolver only finds keys; it never checks a signature. Pass its result to
// identity.Verify (or jws.Verify), which does all of the cryptography and
// succeeds if any one of the keys validates.
package resolvers

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/url"
)

// maxKeys bounds how many keys one subject may publish, so an attacker-
// controlled document can't force unbounded verification work.
const maxKeys = 32

// ErrNoKeys is returned when a subject publishes no usable public key.
var ErrNoKeys = errors.New("no usable public keys found")

// Resolver finds the candidate public keys for a claim's subject.
type Resolver interface {
	// Resolve returns at least one key, or an error. certificate is the
	// claim's DER-encoded X.509 certificate, empty when the claim has none;
	// only SPIFFE resolution reads it.
	Resolve(ctx context.Context, subject string, certificate []byte) ([]crypto.PublicKey, error)
}

// Fetcher retrieves a document over HTTPS. safefetch.Client implements it.
type Fetcher interface {
	Get(ctx context.Context, url string) ([]byte, error)
}

// finish applies the shared post-conditions to a resolver's result.
func finish(keys []crypto.PublicKey, source string) ([]crypto.PublicKey, error) {
	switch {
	case len(keys) == 0:
		return nil, fmt.Errorf("%w at %s", ErrNoKeys, source)
	case len(keys) > maxKeys:
		return nil, fmt.Errorf("%s publishes %d keys, more than the %d allowed", source, len(keys), maxKeys)
	default:
		return keys, nil
	}
}

// toPublicKey keeps the key types jws can verify with, normalizing value
// types to pointers.
func toPublicKey(v any) (crypto.PublicKey, bool) {
	switch k := v.(type) {
	case *ecdsa.PublicKey, *rsa.PublicKey, ed25519.PublicKey:
		return k, true
	case ecdsa.PublicKey:
		return &k, true
	case rsa.PublicKey:
		return &k, true
	default:
		return nil, false
	}
}

// parseHost checks that s is a bare host, optionally with a port, and
// nothing else (no userinfo, path, query, or fragment).
func parseHost(s string) (*url.URL, error) {
	u, err := url.Parse("https://" + s)
	if err != nil || u.Host != s || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid host %q", s)
	}

	return u, nil
}
