// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package didresolver

import (
	"crypto"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/identity/resolvers/internal/claimtest"
	"github.com/agntcy/dir/client/utils/internal/testutil"
	"github.com/multiformats/go-multibase"
	"github.com/stretchr/testify/require"
)

func TestDID_ResolveWeb(t *testing.T) {
	ec := testutil.NewECKey(t, elliptic.P256())
	ed := testutil.NewEdKey(t)

	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/did.json":      didDocumentFor(t, "did:web:acme.com", &ec.PublicKey, ed.Public()),
		"https://acme.com/agents/finance/did.json":   didDocumentFor(t, "did:web:acme.com:agents:finance", &ec.PublicKey),
		"https://acme.com:8443/.well-known/did.json": didDocumentFor(t, "did:web:acme.com%3A8443", ed.Public()),
	}}
	resolver := New(fetcher)

	keys, err := resolver.Resolve(t.Context(), "did:web:acme.com", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&ec.PublicKey, ed.Public()}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:acme.com:agents:finance", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&ec.PublicKey}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:acme.com%3A8443", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{ed.Public()}, keys)
}

func TestDID_ResolveWeb_RejectsMismatchedDocumentID(t *testing.T) {
	// A document that claims to be a different identity must not supply keys.
	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/did.json": didDocumentFor(t, "did:web:other.com", testutil.NewEdKey(t).Public()),
	}}

	_, err := New(fetcher).Resolve(t.Context(), "did:web:acme.com", nil)
	require.ErrorContains(t, err, "does not match subject")
}

func TestDID_ResolveWeb_Errors(t *testing.T) {
	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{
		"https://empty.com/.well-known/did.json":   []byte(`{"id":"did:web:empty.com","verificationMethod":[]}`),
		"https://garbage.com/.well-known/did.json": []byte(`not json`),
		"https://nojwk.com/.well-known/did.json":   []byte(`{"id":"did:web:nojwk.com","verificationMethod":[{"publicKeyMultibase":"z6Mk"}]}`),
	}}
	resolver := New(fetcher)

	_, err := resolver.Resolve(t.Context(), "did:web:empty.com", nil)
	require.ErrorIs(t, err, resolvers.ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "did:web:nojwk.com", nil)
	require.ErrorIs(t, err, resolvers.ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "did:web:garbage.com", nil)
	require.ErrorContains(t, err, "parse DID document")

	_, err = resolver.Resolve(t.Context(), "did:web:missing.com", nil)
	require.ErrorContains(t, err, "fetch DID document")

	_, err = resolver.Resolve(t.Context(), "did:ion:abc", nil)
	require.ErrorContains(t, err, "unsupported did method")
}

func TestDIDWebURL(t *testing.T) {
	valid := map[string]string{
		"did:web:acme.com":                    "https://acme.com/.well-known/did.json",
		"did:web:acme.com:agents:finance":     "https://acme.com/agents/finance/did.json",
		"did:web:acme.com%3A8443":             "https://acme.com:8443/.well-known/did.json",
		"did:web:acme.com%3A8443:agents:bots": "https://acme.com:8443/agents/bots/did.json",
	}

	for subject, want := range valid {
		got, err := didWebURL(subject)
		require.NoError(t, err, subject)
		require.Equal(t, want, got, subject)
	}

	// Anything that could steer the request to a different host or path is rejected.
	invalid := []string{
		"did:web:",
		"did:web::agents",
		"did:web:acme.com:",
		"did:web:acme.com::x",
		"did:web:acme.com:..:etc",
		"did:web:acme.com:%2E%2E:etc",
		"did:web:acme.com:a%2Fb",
		"did:web:acme.com:a%5Cb",
		"did:web:acme.com%2Fpath",
		"did:web:user%40evil.com",
		"did:web:acme.com%3Fq=1",
		"did:web:acme.com%23frag",
		"did:web:acme.com:%zz",
		"did:web:%zz",
	}

	for _, subject := range invalid {
		_, err := didWebURL(subject)
		require.Error(t, err, subject)
	}
}

func TestDID_ResolveKey(t *testing.T) {
	ed := testutil.NewEdKey(t)
	p256 := testutil.NewECKey(t, elliptic.P256())
	p384 := testutil.NewECKey(t, elliptic.P384())
	rsaKey := testutil.NewRSAKey(t)

	rsaDER := x509.MarshalPKCS1PublicKey(&rsaKey.PublicKey)

	cases := map[string]struct {
		code uint64
		raw  []byte
		want crypto.PublicKey
	}{
		"ed25519": {multicodecEd25519Pub, testutil.EdPublic(t, ed), ed.Public()},
		"p256":    {multicodecP256Pub, elliptic.MarshalCompressed(elliptic.P256(), p256.X, p256.Y), &p256.PublicKey}, //nolint:staticcheck // test needs the raw point
		"p384":    {multicodecP384Pub, elliptic.MarshalCompressed(elliptic.P384(), p384.X, p384.Y), &p384.PublicKey}, //nolint:staticcheck // test needs the raw point
		"rsa":     {multicodecRSAPub, rsaDER, &rsaKey.PublicKey},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			subject := "did:key:" + encodeDIDKey(t, tc.code, tc.raw)

			keys, err := New(&testutil.FakeFetcher{}).Resolve(t.Context(), subject, nil)
			require.NoError(t, err)
			testutil.RequireSameKeys(t, []crypto.PublicKey{tc.want}, keys)
		})
	}
}

func TestDID_ResolveKey_DoesNotFetch(t *testing.T) {
	fetcher := &testutil.FakeFetcher{}
	subject := "did:key:" + encodeDIDKey(t, multicodecEd25519Pub, testutil.EdPublic(t, testutil.NewEdKey(t)))

	_, err := New(fetcher).Resolve(t.Context(), subject, nil)
	require.NoError(t, err)
	require.Empty(t, fetcher.Requested)
}

func TestDID_ResolveKey_Errors(t *testing.T) {
	resolver := New(&testutil.FakeFetcher{})

	off := testutil.NewECKey(t, elliptic.P256())
	badPoint := elliptic.MarshalCompressed(elliptic.P256(), off.X, off.Y) //nolint:staticcheck // test needs the raw point
	badPoint[len(badPoint)-1] ^= 0xff                                     // corrupt so the point is (almost surely) off the curve

	subjects := []string{
		"did:key:",
		"did:key:not-multibase",
		"did:key:" + encodeDIDKey(t, 0x9999, []byte("x")),                        // unsupported codec
		"did:key:" + encodeDIDKey(t, multicodecEd25519Pub, []byte("short")),      // wrong length
		"did:key:" + encodeDIDKey(t, multicodecP256Pub, []byte{0x02, 0x01}),      // invalid point
		"did:key:" + encodeDIDKey(t, multicodecRSAPub, []byte("not an rsa key")), // bad DER
	}

	for _, subject := range subjects {
		_, err := resolver.Resolve(t.Context(), subject, nil)
		require.Error(t, err, subject)
	}
}

// didDocumentFor builds a DID document for id holding the given keys.
func didDocumentFor(t *testing.T, id string, keys ...crypto.PublicKey) []byte {
	t.Helper()

	type verificationMethod struct {
		ID           string          `json:"id"`
		Type         string          `json:"type"`
		Controller   string          `json:"controller"`
		PublicKeyJWK json.RawMessage `json:"publicKeyJwk"`
	}

	doc := struct {
		ID                 string               `json:"id"`
		VerificationMethod []verificationMethod `json:"verificationMethod"`
	}{ID: id}

	for _, k := range keys {
		var parsed struct {
			Keys []json.RawMessage `json:"keys"`
		}

		require.NoError(t, json.Unmarshal(testutil.JWKS(t, k), &parsed))
		doc.VerificationMethod = append(doc.VerificationMethod, verificationMethod{
			ID: id + "#key", Type: "JsonWebKey2020", Controller: id, PublicKeyJWK: parsed.Keys[0],
		})
	}

	body, err := json.Marshal(doc)
	require.NoError(t, err)

	return body
}

// encodeDIDKey encodes a multicodec-prefixed key as the base58btc multibase
// string that follows "did:key:".
func encodeDIDKey(t *testing.T, code uint64, raw []byte) string {
	t.Helper()

	encoded, err := multibase.Encode(multibase.Base58BTC, append(binary.AppendUvarint(nil, code), raw...))
	require.NoError(t, err)

	return encoded
}

func TestResolver_SatisfiesContract(t *testing.T) {
	var _ resolvers.Resolver = New(nil)
}

func TestEndToEnd_Web(t *testing.T) {
	key := testutil.NewRSAKey(t)

	resolver := New(&testutil.FakeFetcher{Docs: map[string][]byte{
		"https://acme.com/.well-known/did.json": didDocumentFor(t, "did:web:acme.com", &key.PublicKey),
	}})

	ok, err := claimtest.Verify(t, resolver, claimtest.SignedClaim(t, "did:web:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_Key(t *testing.T) {
	key := testutil.NewEdKey(t)
	subject := "did:key:" + encodeDIDKey(t, multicodecEd25519Pub, testutil.EdPublic(t, key))

	ok, err := claimtest.Verify(t, New(&testutil.FakeFetcher{}), claimtest.SignedClaim(t, subject, key))
	require.NoError(t, err)
	require.True(t, ok)
}
