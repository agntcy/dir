// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package didresolver

import (
	"crypto"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/agntcy/dir/client/utils/identity/resolvers"
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
		"https://acme.com/alice%20smith/did.json":    didDocumentFor(t, "did:web:acme.com:alice%20smith", ed.Public()),
	}}
	resolver := New(fetcher)

	keys, err := resolver.Resolve(t.Context(), "did:web:acme.com", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&ec.PublicKey, ed.Public()}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:acme.com:agents:finance", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&ec.PublicKey}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:acme.com:alice%20smith", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{ed.Public()}, keys)

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

// jwkOf returns key's public JWK as JSON.
func jwkOf(t *testing.T, key crypto.PublicKey) string {
	t.Helper()

	var parsed struct {
		Keys []json.RawMessage `json:"keys"`
	}

	require.NoError(t, json.Unmarshal(testutil.JWKS(t, key), &parsed))

	return string(parsed.Keys[0])
}

func TestDID_ResolveWeb_OnlyAssertionMethods(t *testing.T) {
	ec := testutil.NewECKey(t, elliptic.P256())
	ed := testutil.NewEdKey(t)
	ecJWK, edJWK := jwkOf(t, &ec.PublicKey), jwkOf(t, ed.Public())

	docs := map[string]string{
		// Key material listed for another purpose only, or in no relationship at all.
		"other.com": fmt.Sprintf(`{"id":"did:web:other.com","verificationMethod":[
			{"id":"did:web:other.com#a","publicKeyJwk":%s},{"id":"did:web:other.com#b","publicKeyJwk":%s}],
			"capabilityInvocation":["did:web:other.com#a"],"authentication":["#b"]}`, ecJWK, edJWK),
		// A relative reference, an embedded method, and a method not authorized for assertions.
		"mixed.com": fmt.Sprintf(`{"id":"did:web:mixed.com","verificationMethod":[
			{"id":"#a","publicKeyJwk":%s},{"id":"#b","publicKeyJwk":%s}],
			"assertionMethod":["#a"],"capabilityInvocation":["#b"]}`, ecJWK, edJWK),
		"embedded.com": fmt.Sprintf(`{"id":"did:web:embedded.com","assertionMethod":[
			{"id":"did:web:embedded.com#e","publicKeyJwk":%s}]}`, edJWK),
		// A reference to a method the document doesn't define.
		"dangling.com": `{"id":"did:web:dangling.com","assertionMethod":["#missing"]}`,
	}

	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{}}
	for host, doc := range docs {
		fetcher.Docs["https://"+host+"/.well-known/did.json"] = []byte(doc)
	}

	resolver := New(fetcher)

	for _, host := range []string{"other.com", "dangling.com"} {
		_, err := resolver.Resolve(t.Context(), "did:web:"+host, nil)
		require.ErrorIs(t, err, resolvers.ErrNoKeys, host)
	}

	keys, err := resolver.Resolve(t.Context(), "did:web:mixed.com", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{&ec.PublicKey}, keys)

	keys, err = resolver.Resolve(t.Context(), "did:web:embedded.com", nil)
	require.NoError(t, err)
	testutil.RequireSameKeys(t, []crypto.PublicKey{ed.Public()}, keys)
}

func TestDID_ResolveWeb_Errors(t *testing.T) {
	fetcher := &testutil.FakeFetcher{Docs: map[string][]byte{
		"https://empty.com/.well-known/did.json":   []byte(`{"id":"did:web:empty.com","verificationMethod":[]}`),
		"https://garbage.com/.well-known/did.json": []byte(`not json`),
		"https://nojwk.com/.well-known/did.json":   []byte(`{"id":"did:web:nojwk.com","verificationMethod":[{"id":"#k","publicKeyMultibase":"z6Mk"}],"assertionMethod":["#k"]}`),
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
		"did:web:acme.com:alice%20smith":      "https://acme.com/alice%20smith/did.json", // escaped once, not twice
		"did:web:acme.com:100%25":             "https://acme.com/100%25/did.json",
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
		"did:key:" + encodeDIDKey(t, 0x9999, []byte("x")),                                                                 // unsupported codec
		"did:key:" + encodeDIDKey(t, multicodecEd25519Pub, []byte("short")),                                               // wrong length
		"did:key:" + encodeDIDKey(t, multicodecP256Pub, []byte{0x02, 0x01}),                                               // invalid point
		"did:key:" + encodeDIDKey(t, multicodecRSAPub, []byte("not an rsa key")),                                          // bad DER
		"did:key:" + encodeDIDKey(t, multicodecRSAPub, x509.MarshalPKCS1PublicKey(&testutil.NewSmallRSAKey(t).PublicKey)), // below the RSA floor
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
		AssertionMethod    []string             `json:"assertionMethod"`
	}{ID: id}

	for i, k := range keys {
		var parsed struct {
			Keys []json.RawMessage `json:"keys"`
		}

		require.NoError(t, json.Unmarshal(testutil.JWKS(t, k), &parsed))

		keyID := fmt.Sprintf("%s#key-%d", id, i)
		doc.VerificationMethod = append(doc.VerificationMethod, verificationMethod{
			ID: keyID, Type: "JsonWebKey2020", Controller: id, PublicKeyJWK: parsed.Keys[0],
		})
		doc.AssertionMethod = append(doc.AssertionMethod, keyID)
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

	ok, err := testutil.VerifyClaim(t, resolver, testutil.SignedClaim(t, "did:web:acme.com", key))
	require.NoError(t, err)
	require.True(t, ok)
}

func TestEndToEnd_Key(t *testing.T) {
	key := testutil.NewEdKey(t)
	subject := "did:key:" + encodeDIDKey(t, multicodecEd25519Pub, testutil.EdPublic(t, key))

	ok, err := testutil.VerifyClaim(t, New(&testutil.FakeFetcher{}), testutil.SignedClaim(t, subject, key))
	require.NoError(t, err)
	require.True(t, ok)
}
