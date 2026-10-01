// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknownresolver

import (
	"crypto"
	"crypto/elliptic"
	"encoding/json"
	"testing"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/identity/resolvers/internal/claimtest"
	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
)

func TestWellKnown_Resolve(t *testing.T) {
	ec := testutil.NewECKey(t, elliptic.P256())
	ed := testutil.NewEdKey(t)
	rsaKey := testutil.NewRSAKey(t)
	other := testutil.NewECKey(t, elliptic.P384())

	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/jwks.json":      testutil.JWKS(t, &ec.PublicKey, ed.Public(), &rsaKey.PublicKey),
		"https://acme.com:8443/.well-known/jwks.json": testutil.JWKS(t, &other.PublicKey),
	}}
	resolver := New(fetcher)

	keys, err := resolver.Resolve(t.Context(), "https://acme.com", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&ec.PublicKey, ed.Public(), &rsaKey.PublicKey}, keys)

	// A path on the subject is ignored: the JWKS is always at the host's well-known path.
	keys, err = resolver.Resolve(t.Context(), "https://acme.com/agents/finance", nil)
	require.NoError(t, err)
	require.Len(t, keys, 3)

	keys, err = resolver.Resolve(t.Context(), "https://acme.com:8443", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&other.PublicKey}, keys)
}

func TestWellKnown_Resolve_SkipsEncryptionKeys(t *testing.T) {
	sig := testutil.NewEdKey(t)
	enc := testutil.NewEdKey(t)

	body := []byte(`{"keys":[` +
		string(jwkEntry(t, sig, "sig")) + `,` +
		string(jwkEntry(t, enc, "enc")) +
		`]}`)

	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{"https://acme.com/.well-known/jwks.json": body}}

	keys, err := New(fetcher).Resolve(t.Context(), "https://acme.com", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{sig.Public()}, keys)
}

func TestWellKnown_Resolve_Errors(t *testing.T) {
	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{
		"https://empty.com/.well-known/jwks.json":   []byte(`{"keys":[]}`),
		"https://garbage.com/.well-known/jwks.json": []byte(`<html>not json</html>`),
		"https://symm.com/.well-known/jwks.json":    []byte(`{"keys":[{"kty":"oct","k":"c2VjcmV0"}]}`),
	}}
	resolver := New(fetcher)

	_, err := resolver.Resolve(t.Context(), "https://empty.com", nil)
	require.ErrorIs(t, err, resolvers.ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "https://symm.com", nil) // symmetric keys can't verify
	require.ErrorIs(t, err, resolvers.ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "https://garbage.com", nil)
	require.ErrorContains(t, err, "parse JWKS")

	_, err = resolver.Resolve(t.Context(), "https://missing.com", nil)
	require.ErrorContains(t, err, "fetch JWKS")

	before := len(fetcher.Requested)

	for _, subject := range []string{"acme.com", "http://acme.com", "https://", "https://user@acme.com", "https://acme.com@evil.com", "https://a b", "dns:acme.com"} {
		_, err = resolver.Resolve(t.Context(), subject, nil)
		require.ErrorContains(t, err, "invalid https subject", subject)
	}

	require.Len(t, fetcher.Requested, before, "malformed subjects must not trigger a fetch")
}

// jwkEntry serializes one public key as a JWK with the given "use".
func jwkEntry(t *testing.T, key crypto.Signer, use string) []byte {
	t.Helper()

	pub, err := jwk.FromRaw(key.Public())
	require.NoError(t, err)
	require.NoError(t, pub.Set(jwk.KeyUsageKey, use))

	body, err := json.Marshal(pub)
	require.NoError(t, err)

	return body
}

func TestResolver_SatisfiesContract(t *testing.T) {
	var _ resolvers.Resolver = New(nil)
}

func TestEndToEnd(t *testing.T) {
	key := testutil.NewEdKey(t)

	resolver := New(&testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/jwks.json": testutil.JWKS(t, key.Public()),
	}})

	ok, err := claimtest.Verify(t, resolver, claimtest.SignedClaim(t, "https://acme.com/agents/finance", key))
	require.NoError(t, err)
	require.True(t, ok)
}
