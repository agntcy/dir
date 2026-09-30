// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package resolvers

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"testing"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
)

// fakeFetcher serves canned documents by URL and records what was requested.
type fakeFetcher struct {
	docs      map[string][]byte
	requested []string
}

func (f *fakeFetcher) Get(_ context.Context, url string) ([]byte, error) {
	f.requested = append(f.requested, url)

	doc, ok := f.docs[url]
	if !ok {
		return nil, fmt.Errorf("fetch %s: unexpected status 404", url)
	}

	return doc, nil
}

func newECKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	return key
}

func newEdKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()

	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	return key
}

// edPublic returns the public half of key as an ed25519.PublicKey.
func edPublic(t *testing.T, key ed25519.PrivateKey) ed25519.PublicKey {
	t.Helper()

	pub, ok := key.Public().(ed25519.PublicKey)
	require.True(t, ok)

	return pub
}

func newRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return key
}

func pkcs8PEM(t *testing.T, key crypto.PrivateKey) []byte {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func spkiBase64(t *testing.T, pub crypto.PublicKey) string {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(der)
}

// jwksJSON serializes the public halves of keys as a JWKS document.
func jwksJSON(t *testing.T, keys ...crypto.PublicKey) []byte {
	t.Helper()

	set := jwk.NewSet()

	for _, k := range keys {
		key, err := jwk.FromRaw(k)
		require.NoError(t, err)
		require.NoError(t, set.AddKey(key))
	}

	body, err := json.Marshal(set)
	require.NoError(t, err)

	return body
}

// requireSameKeys asserts got holds exactly the wanted keys, in any order.
func requireSameKeys(t *testing.T, want []crypto.PublicKey, got []crypto.PublicKey) {
	t.Helper()

	require.Len(t, got, len(want))

	for _, w := range want {
		found := false

		for _, g := range got {
			if equaler, ok := w.(interface{ Equal(crypto.PublicKey) bool }); ok && equaler.Equal(g) {
				found = true

				break
			}
		}

		require.True(t, found, "key %T not found in result", w)
	}
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

func pemCertificate(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
