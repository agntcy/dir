// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"crypto"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/json"
	"testing"

	"github.com/multiformats/go-multibase"
	"github.com/stretchr/testify/require"
)

// didDoc builds a DID document for id holding the given keys.
func didDoc(t *testing.T, id string, keys ...crypto.PublicKey) []byte {
	t.Helper()

	type vm struct {
		ID           string          `json:"id"`
		Type         string          `json:"type"`
		Controller   string          `json:"controller"`
		PublicKeyJWK json.RawMessage `json:"publicKeyJwk"`
	}

	doc := struct {
		ID                 string `json:"id"`
		VerificationMethod []vm   `json:"verificationMethod"`
	}{ID: id}

	for _, k := range keys {
		set := jwksJSON(t, k)

		var parsed struct {
			Keys []json.RawMessage `json:"keys"`
		}

		require.NoError(t, json.Unmarshal(set, &parsed))
		doc.VerificationMethod = append(doc.VerificationMethod, vm{
			ID: id + "#key", Type: "JsonWebKey2020", Controller: id, PublicKeyJWK: parsed.Keys[0],
		})
	}

	body, err := json.Marshal(doc)
	require.NoError(t, err)

	return body
}

func TestDID_ResolveWeb(t *testing.T) {
	ec := newECKey(t, elliptic.P256())
	ed := newEdKey(t)

	fetcher := &fakeFetcher{docs: map[string][]byte{
		"https://acme.com/.well-known/did.json":      didDoc(t, "did:web:acme.com", &ec.PublicKey, ed.Public()),
		"https://acme.com/agents/finance/did.json":   didDoc(t, "did:web:acme.com:agents:finance", &ec.PublicKey),
		"https://acme.com:8443/.well-known/did.json": didDoc(t, "did:web:acme.com%3A8443", ed.Public()),
	}}
	resolver := NewDID(fetcher)

	keys, err := resolver.Resolve(t.Context(), "did:web:acme.com", nil)
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{&ec.PublicKey, ed.Public()}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:acme.com:agents:finance", nil)
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{&ec.PublicKey}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:acme.com%3A8443", nil)
	require.NoError(t, err)
	requireSameKeys(t, []crypto.PublicKey{ed.Public()}, keys)
}

func TestDID_ResolveWeb_RejectsMismatchedDocumentID(t *testing.T) {
	// A document that claims to be a different identity must not supply keys.
	fetcher := &fakeFetcher{docs: map[string][]byte{
		"https://acme.com/.well-known/did.json": didDoc(t, "did:web:other.com", newEdKey(t).Public()),
	}}

	_, err := NewDID(fetcher).Resolve(t.Context(), "did:web:acme.com", nil)
	require.ErrorContains(t, err, "does not match subject")
}

func TestDID_ResolveWeb_Errors(t *testing.T) {
	fetcher := &fakeFetcher{docs: map[string][]byte{
		"https://empty.com/.well-known/did.json":   []byte(`{"id":"did:web:empty.com","verificationMethod":[]}`),
		"https://garbage.com/.well-known/did.json": []byte(`not json`),
		"https://nojwk.com/.well-known/did.json":   []byte(`{"id":"did:web:nojwk.com","verificationMethod":[{"publicKeyMultibase":"z6Mk"}]}`),
	}}
	resolver := NewDID(fetcher)

	_, err := resolver.Resolve(t.Context(), "did:web:empty.com", nil)
	require.ErrorIs(t, err, ErrNoKeys)

	_, err = resolver.Resolve(t.Context(), "did:web:nojwk.com", nil)
	require.ErrorIs(t, err, ErrNoKeys)

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
	ed := newEdKey(t)
	p256 := newECKey(t, elliptic.P256())
	p384 := newECKey(t, elliptic.P384())
	rsaKey := newRSAKey(t)

	rsaDER := x509.MarshalPKCS1PublicKey(&rsaKey.PublicKey)

	cases := map[string]struct {
		code uint64
		raw  []byte
		want crypto.PublicKey
	}{
		"ed25519": {multicodecEd25519Pub, edPublic(t, ed), ed.Public()},
		"p256":    {multicodecP256Pub, elliptic.MarshalCompressed(elliptic.P256(), p256.X, p256.Y), &p256.PublicKey}, //nolint:staticcheck // test needs the raw point
		"p384":    {multicodecP384Pub, elliptic.MarshalCompressed(elliptic.P384(), p384.X, p384.Y), &p384.PublicKey}, //nolint:staticcheck // test needs the raw point
		"rsa":     {multicodecRSAPub, rsaDER, &rsaKey.PublicKey},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			subject := "did:key:" + encodeDIDKey(t, tc.code, tc.raw)

			keys, err := NewDID(&fakeFetcher{}).Resolve(t.Context(), subject, nil)
			require.NoError(t, err)
			requireSameKeys(t, []crypto.PublicKey{tc.want}, keys)
		})
	}
}

func TestDID_ResolveKey_DoesNotFetch(t *testing.T) {
	fetcher := &fakeFetcher{}
	subject := "did:key:" + encodeDIDKey(t, multicodecEd25519Pub, edPublic(t, newEdKey(t)))

	_, err := NewDID(fetcher).Resolve(t.Context(), subject, nil)
	require.NoError(t, err)
	require.Empty(t, fetcher.requested)
}

func TestDID_ResolveKey_Errors(t *testing.T) {
	resolver := NewDID(&fakeFetcher{})

	off := newECKey(t, elliptic.P256())
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

// encodeDIDKey encodes a multicodec-prefixed key as a base58btc multibase string.
func encodeDIDKey(t *testing.T, code uint64, raw []byte) string {
	t.Helper()

	prefix := make([]byte, 0, 10)

	for code >= 0x80 {
		prefix = append(prefix, byte(code)|0x80)
		code >>= 7
	}

	prefix = append(prefix, byte(code))

	encoded, err := multibase.Encode(multibase.Base58BTC, append(prefix, raw...))
	require.NoError(t, err)

	return encoded
}
