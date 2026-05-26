// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
)

// canonicalBytes returns the deterministic byte slice that is signed/verified:
// SHA-256( owner_id + ":" + claimed_at ).
func canonicalBytes(ownerID, claimedAt string) []byte {
	h := sha256.Sum256([]byte(ownerID + ":" + claimedAt))

	return h[:]
}

// SignClaimWithKeyFile loads a PEM-encoded EC or RSA private key from keyPath,
// loads the DER/PEM certificate from certPath, validates that the certificate's
// first URI SAN matches claim.OwnerId, then sets claim.Signature and
// claim.Certificate.
func SignClaimWithKeyFile(claim *Claim, keyPath, certPath string) error {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read key file: %w", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("read cert file: %w", err)
	}

	return SignClaim(claim, keyPEM, certPEM)
}

// SignClaim signs the claim using the PEM-encoded private key and certificate.
// It validates that the certificate's first URI SAN equals claim.OwnerId,
// then populates claim.Signature (raw ECDSA/RSA sig) and claim.Certificate (DER).
func SignClaim(claim *Claim, keyPEM, certPEM []byte) error {
	if claim == nil {
		return errors.New("claim is nil")
	}

	// --- parse private key ---
	signer, err := parsePrivateKey(keyPEM)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	// --- parse certificate ---
	cert, certDER, err := parseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	// --- validate identity binding ---
	if err := validateCertIdentity(cert, claim.GetOwnerId()); err != nil {
		return err
	}

	// --- sign ---
	digest := canonicalBytes(claim.GetOwnerId(), claim.GetClaimedAt())

	sig, err := signer.Sign(rand.Reader, digest, crypto.SHA256)
	if err != nil {
		return fmt.Errorf("sign claim: %w", err)
	}

	claim.Signature = sig
	claim.Certificate = certDER

	return nil
}

// VerifyClaim verifies claim.Signature against claim.Certificate and optionally
// against a SPIFFE trust bundle.
//
// Verification steps:
//  1. Decode and parse claim.Certificate (DER).
//  2. If trustedCerts is non-nil, verify the certificate chains to one of them.
//  3. Confirm the certificate's first URI SAN equals claim.OwnerId.
//  4. Verify the ECDSA/RSA signature over canonicalBytes(owner_id, claimed_at).
func VerifyClaim(claim *Claim, trustedCerts []*x509.Certificate) error {
	if claim == nil {
		return errors.New("claim is nil")
	}

	if len(claim.GetSignature()) == 0 {
		return errors.New("claim has no signature")
	}

	if len(claim.GetCertificate()) == 0 {
		return errors.New("claim has no certificate")
	}

	cert, err := x509.ParseCertificate(claim.GetCertificate())
	if err != nil {
		return fmt.Errorf("parse claim certificate: %w", err)
	}

	// Optional trust bundle verification.
	if len(trustedCerts) > 0 {
		pool := x509.NewCertPool()
		for _, ca := range trustedCerts {
			pool.AddCert(ca)
		}

		if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
			return fmt.Errorf("certificate not trusted: %w", err)
		}
	}

	// Identity binding: URI SAN must equal owner_id.
	if err := validateCertIdentity(cert, claim.GetOwnerId()); err != nil {
		return err
	}

	// Signature verification.
	digest := canonicalBytes(claim.GetOwnerId(), claim.GetClaimedAt())

	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, digest, claim.GetSignature()) {
			return errors.New("ECDSA signature verification failed")
		}

	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest, claim.GetSignature()); err != nil {
			return fmt.Errorf("RSA signature verification failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported public key type: %T", cert.PublicKey)
	}

	return nil
}

// IsSigned returns true when the claim carries both a signature and a certificate.
func IsSigned(claim *Claim) bool {
	return claim != nil && len(claim.GetSignature()) > 0 && len(claim.GetCertificate()) > 0
}

// validateCertIdentity checks that the certificate's first URI SAN equals ownerID.
func validateCertIdentity(cert *x509.Certificate, ownerID string) error {
	if len(cert.URIs) == 0 {
		return errors.New("certificate has no URI SAN")
	}

	certURI := cert.URIs[0].String()
	if certURI != ownerID {
		return fmt.Errorf("certificate URI SAN %q does not match owner_id %q", certURI, ownerID)
	}

	return nil
}

// parsePrivateKey decodes a PEM block and returns a crypto.Signer.
// Supports EC PRIVATE KEY, RSA PRIVATE KEY, and PRIVATE KEY (PKCS8).
func parsePrivateKey(pemBytes []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block from key file")
	}

	switch block.Type {
	case "EC PRIVATE KEY":
		key, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse EC private key: %w", err)
		}

		// Verify the key is on a supported curve.
		if key.Curve != elliptic.P256() && key.Curve != elliptic.P384() && key.Curve != elliptic.P521() {
			return nil, fmt.Errorf("unsupported EC curve: %v", key.Curve)
		}

		return key, nil

	case "RSA PRIVATE KEY":
		key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse RSA private key: %w", err)
		}

		return key, nil

	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
		}

		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key type %T does not implement crypto.Signer", key)
		}

		return signer, nil

	default:
		return nil, fmt.Errorf("unsupported PEM block type: %q", block.Type)
	}
}

// parseCertificate decodes a PEM or raw DER certificate and returns the parsed
// *x509.Certificate together with its raw DER bytes.
func parseCertificate(data []byte) (*x509.Certificate, []byte, error) {
	var derBytes []byte

	if block, _ := pem.Decode(data); block != nil {
		derBytes = block.Bytes
	} else {
		derBytes = data
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse certificate: %w", err)
	}

	// Validate SPIFFE URI SAN exists.
	if len(cert.URIs) == 0 {
		return nil, nil, errors.New("certificate contains no URI SAN (required for SPIFFE identity)")
	}

	for _, uri := range cert.URIs {
		if err := validateSpiffeURI(uri); err != nil {
			return nil, nil, err
		}
	}

	return cert, derBytes, nil
}

// validateSpiffeURI checks that a URI follows the spiffe:// scheme.
func validateSpiffeURI(u *url.URL) error {
	if u.Scheme != "spiffe" {
		return fmt.Errorf("URI SAN %q is not a SPIFFE ID (must use spiffe:// scheme)", u.String())
	}

	return nil
}
