// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"crypto"
	"crypto/elliptic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWellKnown_Resolve(t *testing.T) {
	ec := newECKey(t, elliptic.P256())
	ed := newEdKey(t)
	rsaKey := newRSAKey(t)
	other := newECKey(t, elliptic.P384())

	fetcher := &fakeFetcher{docs: map[string][]byte{
		"https://acme.com/.well-known/jwks.json":      jwksJSON(t, &ec.PublicKey, ed.Public(), &rsaKey.PublicKey),
		"https://acme.com:8443/.well-known/jwks.json": jwksJSON(t, &other.PublicKey),
	}}
	resolver := NewWellKnown(fetcher)

	keys, err := resolver.Resolve(t.Context(), "https://acme.com", nil)
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{&ec.PublicKey, ed.Public(), &rsaKey.PublicKey}, keys)

	// A path on the subject is ignored: the JWKS is always at the host's well-known path.
	keys, err = resolver.Resolve(t.Context(), "https://acme.com/agents/finance", nil)
	require.NoError(t, err)
	require.Len(t, keys, 3)

	keys, err = resolver.Resolve(t.Context(), "https://acme.com:8443", nil)
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{&other.PublicKey}, keys)
}

func TestWellKnown_Resolve_SkipsEncryptionKeys(t *testing.T) {
	sig := newEdKey(t)
	enc := newEdKey(t)

	body := []byte(`{"keys":[` +
		string(jwkEntry(t, sig, "sig")) + `,` +
		string(jwkEntry(t, enc, "enc")) +
		`]}`)

	fetcher := &fakeFetcher{docs: map[string][]byte{"https://acme.com/.well-known/jwks.json": body}}

	keys, err := NewWellKnown(fetcher).Resolve(t.Context(), "https://acme.com", nil)
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{sig.Public()}, keys)
}

func TestWellKnown_Resolve_Errors(t *testing.T) {
	fetcher := &fakeFetcher{docs: map[string][]byte{
		"https://empty.com/.well-known/jwks.json":   []byte(`{"keys":[]}`),
		"https://garbage.com/.well-known/jwks.json": []byte(`<html>not json</html>`),
		"https://symm.com/.well-known/jwks.json":    []byte(`{"keys":[{"kty":"oct","k":"c2VjcmV0"}]}`),
	}}
	resolver := NewWellKnown(fetcher)

	_, err := resolver.Resolve(t.Context(), "https://empty.com", nil)
	require.ErrorIs(t, err, ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "https://symm.com", nil) // symmetric keys can't verify
	require.ErrorIs(t, err, ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "https://garbage.com", nil)
	require.ErrorContains(t, err, "parse JWKS")

	_, err = resolver.Resolve(t.Context(), "https://missing.com", nil)
	require.ErrorContains(t, err, "fetch JWKS")

	before := len(fetcher.requested)

	for _, subject := range []string{"acme.com", "http://acme.com", "https://", "https://user@acme.com", "https://acme.com@evil.com", "https://a b", "dns:acme.com"} {
		_, err = resolver.Resolve(t.Context(), subject, nil)
		require.ErrorContains(t, err, "invalid https subject", subject)
	}

	require.Len(t, fetcher.requested, before, "malformed subjects must not trigger a fetch")
}
