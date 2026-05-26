// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/url"
	"testing"
	"time"

	ownershipv1 "github.com/agntcy/dir/api/ownership/v1"
)

const testSpiffeID = "spiffe://example.org/agent/test"

// generateTestKeyCert returns PEM-encoded EC private key and self-signed cert
// whose URI SAN is set to spiffeID.
func generateTestKeyCert(t *testing.T, spiffeID string) ([]byte, []byte) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	spiffeURI, err := url.Parse(spiffeID)
	if err != nil {
		t.Fatalf("parse SPIFFE URI: %v", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		URIs:         []*url.URL{spiffeURI},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal EC key: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privDER})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return keyPEM, certPEM
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	keyPEM, certPEM := generateTestKeyCert(t, testSpiffeID)

	claim := &ownershipv1.Claim{
		OwnerId:   testSpiffeID,
		ClaimedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := ownershipv1.SignClaim(claim, keyPEM, certPEM); err != nil {
		t.Fatalf("SignClaim: %v", err)
	}

	if !ownershipv1.IsSigned(claim) {
		t.Fatal("IsSigned should be true after signing")
	}

	if err := ownershipv1.VerifyClaim(claim, nil); err != nil {
		t.Fatalf("VerifyClaim: %v", err)
	}
}

func TestVerifyClaim_TamperedSignature(t *testing.T) {
	keyPEM, certPEM := generateTestKeyCert(t, testSpiffeID)

	claim := &ownershipv1.Claim{
		OwnerId:   testSpiffeID,
		ClaimedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := ownershipv1.SignClaim(claim, keyPEM, certPEM); err != nil {
		t.Fatalf("SignClaim: %v", err)
	}

	// Flip a byte in the signature to corrupt it.
	claim.Signature[0] ^= 0xFF

	if err := ownershipv1.VerifyClaim(claim, nil); err == nil {
		t.Fatal("VerifyClaim should fail with a corrupted signature")
	}
}

func TestVerifyClaim_TamperedOwnerID(t *testing.T) {
	keyPEM, certPEM := generateTestKeyCert(t, testSpiffeID)

	claim := &ownershipv1.Claim{
		OwnerId:   testSpiffeID,
		ClaimedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := ownershipv1.SignClaim(claim, keyPEM, certPEM); err != nil {
		t.Fatalf("SignClaim: %v", err)
	}

	// Alter owner_id after signing — cert SAN still matches original but
	// the canonical bytes will differ, so signature check should fail.
	claim.OwnerId = "spiffe://example.org/agent/attacker"

	if err := ownershipv1.VerifyClaim(claim, nil); err == nil {
		t.Fatal("VerifyClaim should fail when owner_id is altered after signing")
	}
}

func TestSignClaim_IdentityMismatch(t *testing.T) {
	keyPEM, certPEM := generateTestKeyCert(t, "spiffe://example.org/agent/alice")

	claim := &ownershipv1.Claim{
		OwnerId:   "spiffe://example.org/agent/bob", // different from cert
		ClaimedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if err := ownershipv1.SignClaim(claim, keyPEM, certPEM); err == nil {
		t.Fatal("SignClaim should fail when cert URI SAN does not match owner_id")
	}
}

func TestVerifyClaim_UnsignedClaim(t *testing.T) {
	claim := &ownershipv1.Claim{
		OwnerId:   testSpiffeID,
		ClaimedAt: time.Now().UTC().Format(time.RFC3339),
	}

	if ownershipv1.IsSigned(claim) {
		t.Fatal("IsSigned should be false for an unsigned claim")
	}

	// VerifyClaim on an unsigned claim must return an error.
	if err := ownershipv1.VerifyClaim(claim, nil); err == nil {
		t.Fatal("VerifyClaim should fail for an unsigned claim (no signature)")
	}
}
