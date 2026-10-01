// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package testutil holds test fixtures shared by the jws, identity and
// resolver packages: key generators, PEM and certificate helpers, a fake
// fetcher, key assertions, and the sign-resolve-verify claim flow. It imports
// jws and identity, so their tests must be external (package jws_test).
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
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/agntcy/dir/client/utils/identity"
	"github.com/agntcy/dir/client/utils/identity/resolvers"
	"github.com/agntcy/dir/client/utils/jws"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"
)

const (
	rsaKeyBits      = 2048
	smallRSAKeyBits = 1024

	// SVIDSubject is the SPIFFE ID the test certificates carry by default.
	SVIDSubject = "spiffe://acme.com/agents/finance"

	// RecordCID is the record every test claim is attached to.
	RecordCID = "bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
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

// NewKey generates a signing key of the given kind: ES256, ES384, EdDSA or RS256.
func NewKey(t *testing.T, kind string) crypto.Signer {
	t.Helper()

	switch kind {
	case "ES256":
		return NewECKey(t, elliptic.P256())
	case "ES384":
		return NewECKey(t, elliptic.P384())
	case "EdDSA":
		return NewEdKey(t)
	case "RS256":
		return NewRSAKey(t)
	default:
		t.Fatalf("unsupported key kind %q", kind)

		return nil
	}
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

// NewSmallRSAKey generates a 1024-bit RSA key, below the size jws accepts.
func NewSmallRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, smallRSAKeyBits) //nolint:gosec // deliberately weak, to test the size floor
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

// SelfSignedCertPEM returns a self-signed certificate for key carrying
// uriSAN as its only URI SAN.
func SelfSignedCertPEM(t *testing.T, key crypto.Signer, uriSAN string) []byte {
	t.Helper()

	uri, err := url.Parse(uriSAN)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: uriSAN},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{uri},
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, key.Public(), key)
	require.NoError(t, err)

	return CertificatePEM(der)
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

// SignedClaim signs a fresh identity claim for subject the way a publisher would.
func SignedClaim(t *testing.T, subject string, key crypto.PrivateKey) *identityv1.Claim {
	t.Helper()

	signer, err := jws.NewKeySigner(PKCS8PEM(t, key), nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: subject}
	require.NoError(t, identity.Sign(claim, RecordCID, signer))

	return claim
}

// SignedCertClaim signs a claim for subject that carries certPEM, the certificate of key.
func SignedCertClaim(t *testing.T, subject string, key crypto.PrivateKey, certPEM []byte) *identityv1.Claim {
	t.Helper()

	signer, err := jws.NewKeySigner(PKCS8PEM(t, key), nil)
	require.NoError(t, err)

	claim := &identityv1.Claim{Role: identityv1.ClaimRole_CLAIM_ROLE_IDENTITY, Subject: subject}
	require.NoError(t, identity.Sign(claim, RecordCID, signer, identity.WithCertificate(certPEM)))

	return claim
}

// VerifyClaim is the flow a real caller follows: resolve the subject's keys, then
// verify the claim against all of them.
func VerifyClaim(t *testing.T, r resolvers.Resolver, claim *identityv1.Claim) (bool, error) {
	t.Helper()

	var certificate []byte

	if claim.Certificate != nil {
		var err error

		certificate, err = base64.StdEncoding.DecodeString(claim.GetCertificate())
		require.NoError(t, err)
	}

	keys, err := r.Resolve(t.Context(), claim.GetSubject(), certificate)
	if err != nil {
		return false, fmt.Errorf("resolve: %w", err)
	}

	ok, err := identity.Verify(claim, RecordCID, claim.GetSubject(), keys...)
	if err != nil {
		return false, fmt.Errorf("verify: %w", err)
	}

	return ok, nil
}
