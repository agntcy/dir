// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package keyutil holds the helpers the individual resolver packages share.
package keyutil

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"fmt"
	"net/url"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// MaxKeys bounds how many keys one subject may publish, so an attacker-
// controlled document can't force unbounded verification work.
const MaxKeys = 32

// Finish applies the shared post-conditions to a resolver's result: at least
// one key and no more than MaxKeys. source names where the keys came from.
func Finish(keys []crypto.PublicKey, source string) ([]crypto.PublicKey, error) {
	switch {
	case len(keys) == 0:
		return nil, fmt.Errorf("%w at %s", resolvers.ErrNoKeys, source)
	case len(keys) > MaxKeys:
		return nil, fmt.Errorf("%s publishes %d keys, more than the %d allowed", source, len(keys), MaxKeys)
	default:
		return keys, nil
	}
}

// ToPublicKey keeps the key types jws can verify with, normalizing value
// types to pointers.
func ToPublicKey(v any) (crypto.PublicKey, bool) {
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

// ParseHost checks that s is a bare host, optionally with a port, and
// nothing else (no userinfo, path, query, or fragment).
func ParseHost(s string) (*url.URL, error) {
	u, err := url.Parse("https://" + s)
	if err != nil || u.Host != s || u.Hostname() == "" {
		return nil, fmt.Errorf("invalid host %q", s)
	}

	return u, nil
}

// PublicKeyFromJWK returns the signature-verification public key of key,
// skipping encryption keys and key types jws can't verify with.
func PublicKeyFromJWK(key jwk.Key) (crypto.PublicKey, bool) {
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

	return ToPublicKey(raw)
}
