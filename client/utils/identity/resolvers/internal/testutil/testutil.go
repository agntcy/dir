// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package testutil holds fixtures shared by the resolver packages' tests:
// key generators, document builders, a fake fetcher, and a test CA.
package testutil

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/multiformats/go-multibase"
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/require"
)

const (
	rsaKeyBits = 2048
	svidSerial = 2
)

// FakeFetcher serves canned documents by URL and records what was requested.
type FakeFetcher struct {
	Docs      map[string][]byte
	Requested []string
}

// Get implements resolvers.Fetcher.
func (f *FakeFetcher) Get(_ context.Context, url string) ([]byte, error) {
	f.Requested = append(f.Requested, url)

	doc, ok := f.Docs[url]
	if !ok {
		return nil, fmt.Errorf("fetch %s: unexpected status 404", url)
	}

	return doc, nil
}

// NewECKey generates an ECDSA key on curve.
func NewECKey(t *testing.T, curve elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	require.NoError(t, err)

	return key
}

// NewEdKey generates an Ed25519 key.
func NewEdKey(t *testing.T) ed25519.PrivateKey {
	t.Helper()

	_, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	return key
}

// EdPublic returns the public half of key as an ed25519.PublicKey.
func EdPublic(t *testing.T, key ed25519.PrivateKey) ed25519.PublicKey {
	t.Helper()

	pub, ok := key.Public().(ed25519.PublicKey)
	require.True(t, ok)

	return pub
}

// NewRSAKey generates a 2048-bit RSA key.
func NewRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	require.NoError(t, err)

	return key
}

// PKCS8PEM encodes key as an unencrypted PKCS#8 PEM block.
func PKCS8PEM(t *testing.T, key crypto.PrivateKey) []byte {
	t.Helper()

	der, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// CertificatePEM encodes a DER certificate as a PEM block.
func CertificatePEM(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

// SPKIBase64 encodes pub as base64 DER SubjectPublicKeyInfo.
func SPKIBase64(t *testing.T, pub crypto.PublicKey) string {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(pub)
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(der)
}

// JWKS serializes the public halves of keys as a JWKS document.
func JWKS(t *testing.T, keys ...crypto.PublicKey) []byte {
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

// JWKEntry serializes one public key as a JWK with the given "use".
func JWKEntry(t *testing.T, key crypto.Signer, use string) []byte {
	t.Helper()

	pub, err := jwk.FromRaw(key.Public())
	require.NoError(t, err)
	require.NoError(t, pub.Set(jwk.KeyUsageKey, use))

	body, err := json.Marshal(pub)
	require.NoError(t, err)

	return body
}

// DIDDocument builds a DID document for id holding the given keys.
func DIDDocument(t *testing.T, id string, keys ...crypto.PublicKey) []byte {
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

		require.NoError(t, json.Unmarshal(JWKS(t, k), &parsed))
		doc.VerificationMethod = append(doc.VerificationMethod, verificationMethod{
			ID: id + "#key", Type: "JsonWebKey2020", Controller: id, PublicKeyJWK: parsed.Keys[0],
		})
	}

	body, err := json.Marshal(doc)
	require.NoError(t, err)

	return body
}

// RequireSameKeys asserts got holds exactly the wanted keys, in any order.
func RequireSameKeys(t *testing.T, want, got []crypto.PublicKey) {
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

// CA is a throwaway certificate authority for issuing test SVIDs.
type CA struct {
	Cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

// NewCA creates a self-signed CA.
func NewCA(t *testing.T, name string) *CA {
	t.Helper()

	key := NewECKey(t, elliptic.P256())
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &CA{Cert: cert, key: key}
}

// SVIDOptions shapes a certificate issued by CA.Issue.
type SVIDOptions struct {
	URIs      []string
	NotBefore time.Time
	NotAfter  time.Time
	IsCA      bool
}

// Issue returns the DER of a certificate for key signed by the CA. By
// default it is valid from a minute ago for an hour.
func (ca *CA) Issue(t *testing.T, key crypto.Signer, opts SVIDOptions) []byte {
	t.Helper()

	if opts.NotBefore.IsZero() {
		opts.NotBefore = time.Now().Add(-time.Minute)
	}

	if opts.NotAfter.IsZero() {
		opts.NotAfter = time.Now().Add(time.Hour)
	}

	uris := make([]*url.URL, 0, len(opts.URIs))

	for _, u := range opts.URIs {
		parsed, err := url.Parse(u)
		require.NoError(t, err)

		uris = append(uris, parsed)
	}

	usage := x509.KeyUsageDigitalSignature
	if opts.IsCA {
		usage = x509.KeyUsageCertSign
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(svidSerial),
		Subject:               pkix.Name{CommonName: "svid"},
		NotBefore:             opts.NotBefore,
		NotAfter:              opts.NotAfter,
		URIs:                  uris,
		KeyUsage:              usage,
		IsCA:                  opts.IsCA,
		BasicConstraintsValid: true,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.Cert, key.Public(), ca.key)
	require.NoError(t, err)

	return der
}

// Bundle returns a trust bundle set for trustDomain rooted at cas.
func Bundle(t *testing.T, trustDomain string, cas ...*CA) *x509bundle.Set {
	t.Helper()

	td, err := spiffeid.TrustDomainFromString(trustDomain)
	require.NoError(t, err)

	certs := make([]*x509.Certificate, 0, len(cas))
	for _, ca := range cas {
		certs = append(certs, ca.Cert)
	}

	return x509bundle.NewSet(x509bundle.FromX509Authorities(td, certs))
}

// EncodeDIDKey encodes a multicodec-prefixed key as the base58btc multibase
// string that follows "did:key:".
func EncodeDIDKey(t *testing.T, code uint64, raw []byte) string {
	t.Helper()

	encoded, err := multibase.Encode(multibase.Base58BTC, append(binary.AppendUvarint(nil, code), raw...))
	require.NoError(t, err)

	return encoded
}
