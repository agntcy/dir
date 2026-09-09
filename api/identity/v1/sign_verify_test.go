// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"testing"
	"time"

	identityv1 "github.com/agntcy/dir/api/identity/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSpiffeID = "spiffe://acme.com/agents/finance"

// generateTestKeyPEM returns a PEM-encoded ECDSA P-256 private key.
func generateTestKeyPEM(t *testing.T) []byte {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
}

// generateTestSpiffeKeyCert returns a PEM-encoded ECDSA key, a self-signed
// certificate whose URI SAN is set to spiffeID, and the parsed certificate
// (usable as its own trust anchor, since it's self-signed).
func generateTestSpiffeKeyCert(t *testing.T, spiffeID string) (keyPEM, certPEM []byte, cert *x509.Certificate) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	spiffeURI, err := url.Parse(spiffeID)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{spiffeURI},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	require.NoError(t, err)

	parsedCert, err := x509.ParseCertificate(derBytes)
	require.NoError(t, err)

	privDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err)

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return keyPEM, certPEM, parsedCert
}

// fakeResolver is a controllable identityv1.KeyResolver for testing the
// dns/https/did verification path without a real server/identity registry.
type fakeResolver struct {
	verified bool
	err      error
}

func (f *fakeResolver) Verify(_ context.Context, _ string, _, _ []byte) (bool, error) {
	return f.verified, f.err
}

var _ identityv1.KeyResolver = (*fakeResolver)(nil)

func TestSignVerifyIdentityClaim_KeySigner_RoundTrip(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)
	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim("did:web:acme.com:agents:finance")
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	assert.NotEmpty(t, claim.GetSignature())
	assert.NotEmpty(t, claim.GetSignedAt())
	assert.Empty(t, claim.GetCertificate(), "key-based claims must not embed a certificate")

	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", "did:web:acme.com:agents:finance", &fakeResolver{verified: true}, nil)
	assert.True(t, result.Verified, result.Error)
	assert.Equal(t, "did", result.SubjectType)
}

func TestSignVerifyIdentityClaim_KeySigner_Ed25519RoundTrip(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	der, err := x509.MarshalPKCS8PrivateKey(priv)
	require.NoError(t, err)

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim("did:key:z6MkTest")
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", "did:key:z6MkTest", &fakeResolver{verified: true}, nil)
	assert.True(t, result.Verified, result.Error)
}

func TestSignVerifyOwnershipClaim_SpiffeSigner_RoundTrip(t *testing.T) {
	keyPEM, certPEM, cert := generateTestSpiffeKeyCert(t, testSpiffeID)
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewOwnershipClaim(testSpiffeID)
	require.NoError(t, identityv1.SignOwnershipClaim(claim, "baguqeera-cid", signer))

	assert.NotEmpty(t, claim.GetCertificate(), "spiffe claims must embed a certificate")

	result := identityv1.VerifyOwnershipClaim(t.Context(), claim, "baguqeera-cid", testSpiffeID, nil, []*x509.Certificate{cert})
	assert.True(t, result.Verified, result.Error)
	assert.Equal(t, "spiffe", result.SubjectType)
}

func TestVerifyClaim_SpiffeFailsWithoutTrustBundle(t *testing.T) {
	keyPEM, certPEM, _ := generateTestSpiffeKeyCert(t, testSpiffeID)
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewOwnershipClaim(testSpiffeID)
	require.NoError(t, identityv1.SignOwnershipClaim(claim, "baguqeera-cid", signer))

	// No trust bundle configured for this subject's trust domain: a
	// self-signed (or otherwise attacker-suppliable) certificate alone must
	// not be enough to verify a SPIFFE claim.
	result := identityv1.VerifyOwnershipClaim(t.Context(), claim, "baguqeera-cid", testSpiffeID, nil, nil)
	assert.False(t, result.Verified)
	assert.Contains(t, result.Error, "trust bundle")
}

func TestVerifyClaim_FailsWhenRecordCIDChanges(t *testing.T) {
	keyPEM, certPEM, cert := generateTestSpiffeKeyCert(t, testSpiffeID)
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim(testSpiffeID)
	require.NoError(t, identityv1.SignIdentityClaim(claim, "cid-A", signer))

	// Replaying the same signed referrer against a different record's CID
	// must not verify - this is the fix over PR #1478's CID-less payload.
	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "cid-B", testSpiffeID, nil, []*x509.Certificate{cert})
	assert.False(t, result.Verified)
}

func TestVerifyClaim_FailsWhenSubjectTampered(t *testing.T) {
	keyPEM, certPEM, cert := generateTestSpiffeKeyCert(t, testSpiffeID)
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim(testSpiffeID)
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	claim.Subject = "spiffe://acme.com/agents/attacker"

	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", testSpiffeID, nil, []*x509.Certificate{cert})
	assert.False(t, result.Verified)
}

func TestVerifyClaim_FailsWhenSubjectDoesNotMatchRecordAnnotation(t *testing.T) {
	keyPEM, certPEM, cert := generateTestSpiffeKeyCert(t, testSpiffeID)
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim(testSpiffeID)
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	// The claim is validly signed for this record's CID, but the record
	// itself declares a different identity - e.g. someone else's otherwise
	// valid claim being attached to this record.
	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", "spiffe://acme.com/agents/someone-else", nil, []*x509.Certificate{cert})
	assert.False(t, result.Verified)
	assert.Contains(t, result.Error, "does not match")
}

func TestVerifyClaim_FailsWhenRecordHasNoDeclaredSubject(t *testing.T) {
	keyPEM, certPEM, cert := generateTestSpiffeKeyCert(t, testSpiffeID)
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim(testSpiffeID)
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", "", nil, []*x509.Certificate{cert})
	assert.False(t, result.Verified)
	assert.Contains(t, result.Error, "does not declare")
}

func TestVerifyClaim_FailsWhenExpired(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)
	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim("did:web:acme.com:agents:finance")
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	expired := time.Now().Add(-time.Hour).Format(time.RFC3339)
	claim.ExpiresAt = &expired

	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", "did:web:acme.com:agents:finance", &fakeResolver{verified: true}, nil)
	assert.False(t, result.Verified)
	assert.Contains(t, result.Error, "expired")
}

func TestSignOwnershipClaim_SpiffeSubjectMismatch(t *testing.T) {
	keyPEM, certPEM, _ := generateTestSpiffeKeyCert(t, "spiffe://acme.com/agents/alice")
	signer, err := identityv1.NewSpiffeSigner(keyPEM, certPEM)
	require.NoError(t, err)

	// Claim asserts a different subject than the certificate's SPIFFE ID.
	claim := identityv1.NewOwnershipClaim("spiffe://acme.com/agents/bob")

	err = identityv1.SignOwnershipClaim(claim, "baguqeera-cid", signer)
	assert.Error(t, err)
}

func TestVerifyClaim_ResolverRejection(t *testing.T) {
	keyPEM := generateTestKeyPEM(t)
	signer, err := identityv1.NewKeySigner(keyPEM)
	require.NoError(t, err)

	claim := identityv1.NewIdentityClaim("https://acme.com/agent")
	require.NoError(t, identityv1.SignIdentityClaim(claim, "baguqeera-cid", signer))

	result := identityv1.VerifyIdentityClaim(t.Context(), claim, "baguqeera-cid", "https://acme.com/agent", &fakeResolver{verified: false}, nil)
	assert.False(t, result.Verified)
}

func TestVerifyClaim_NilClaim(t *testing.T) {
	result := identityv1.VerifyIdentityClaim(t.Context(), nil, "baguqeera-cid", "https://acme.com/agent", nil, nil)
	assert.False(t, result.Verified)
	assert.NotEmpty(t, result.Error)
}
