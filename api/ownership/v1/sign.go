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
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
)

const (
	// OwnershipContentType is the content_type value that marks a Signature referrer
	// as a SPIFFE-based ownership claim (mirrors corev1.OwnershipContentType).
	OwnershipContentType = corev1.OwnershipContentType

	ownerIDAnnotation = "owner_id"
	ownerAlgorithmEC  = "ECDSA_SHA256"
	ownerAlgorithmRSA = "RSA_SHA256"
)

// IsOwnershipClaim returns true when the Signature referrer carries an ownership claim.
func IsOwnershipClaim(sig *signv1.Signature) bool {
	return sig != nil && sig.GetContentType() == OwnershipContentType
}

// NewClaim builds an unsigned ownership Signature with content_type and owner_id set.
// Call SignClaim / SignClaimWithKeyFile to attach a cryptographic proof.
func NewClaim(ownerID string) *signv1.Signature {
	return &signv1.Signature{
		ContentType: OwnershipContentType,
		Annotations: map[string]string{ownerIDAnnotation: ownerID},
		SignedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}

// SignClaimWithKeyFile loads PEM key and cert from disk and calls SignClaim.
func SignClaimWithKeyFile(sig *signv1.Signature, keyPath, certPath string) error {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("read key file: %w", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("read cert file: %w", err)
	}

	return SignClaim(sig, keyPEM, certPEM)
}

// SignClaim signs the ownership Signature using the PEM-encoded private key and certificate.
// It validates that the certificate's first URI SAN equals the owner_id annotation,
// then sets sig.Algorithm, sig.Signature (Base64), and sig.Certificate (Base64 DER).
func SignClaim(sig *signv1.Signature, keyPEM, certPEM []byte) error {
	if sig == nil {
		return errors.New("signature is nil")
	}

	ownerID := sig.GetAnnotations()[ownerIDAnnotation]
	if ownerID == "" {
		return errors.New("owner_id annotation is required before signing")
	}

	signer, err := parsePrivateKey(keyPEM)
	if err != nil {
		return fmt.Errorf("parse private key: %w", err)
	}

	cert, certDER, err := parseCertificate(certPEM)
	if err != nil {
		return fmt.Errorf("parse certificate: %w", err)
	}

	if err := validateCertIdentity(cert, ownerID); err != nil {
		return err
	}

	digest := canonicalBytes(ownerID, sig.GetSignedAt())

	rawSig, err := signer.Sign(rand.Reader, digest, crypto.SHA256)
	if err != nil {
		return fmt.Errorf("sign claim: %w", err)
	}

	switch signer.Public().(type) {
	case *ecdsa.PublicKey:
		sig.Algorithm = ownerAlgorithmEC
	case *rsa.PublicKey:
		sig.Algorithm = ownerAlgorithmRSA
	}

	sig.Signature = base64.StdEncoding.EncodeToString(rawSig)
	sig.Certificate = base64.StdEncoding.EncodeToString(certDER)

	return nil
}

// VerifyClaim verifies the cryptographic proof in an ownership Signature.
//
// Steps:
//  1. Decode sig.Certificate (Base64 DER) and parse the X.509 certificate.
//  2. If trustedCerts is non-nil, verify the cert chains to one of them.
//  3. Confirm the cert's first URI SAN equals the owner_id annotation.
//  4. Verify the ECDSA/RSA signature over canonicalBytes(owner_id, signed_at).
func VerifyClaim(sig *signv1.Signature, trustedCerts []*x509.Certificate) error {
	if sig == nil {
		return errors.New("signature is nil")
	}

	if sig.GetSignature() == "" {
		return errors.New("signature is empty")
	}

	if sig.GetCertificate() == "" {
		return errors.New("certificate is empty")
	}

	certDER, err := base64.StdEncoding.DecodeString(sig.GetCertificate())
	if err != nil {
		return fmt.Errorf("base64-decode certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return fmt.Errorf("parse claim certificate: %w", err)
	}

	if len(trustedCerts) > 0 {
		pool := x509.NewCertPool()
		for _, ca := range trustedCerts {
			pool.AddCert(ca)
		}

		if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
			return fmt.Errorf("certificate not trusted: %w", err)
		}
	}

	ownerID := sig.GetAnnotations()[ownerIDAnnotation]

	if err := validateCertIdentity(cert, ownerID); err != nil {
		return err
	}

	rawSig, err := base64.StdEncoding.DecodeString(sig.GetSignature())
	if err != nil {
		return fmt.Errorf("base64-decode signature: %w", err)
	}

	digest := canonicalBytes(ownerID, sig.GetSignedAt())

	return verifySignatureBytes(cert, digest, rawSig)
}

// verifySignatureBytes verifies rawSig over digest using the public key from cert.
func verifySignatureBytes(cert *x509.Certificate, digest, rawSig []byte) error {
	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if !ecdsa.VerifyASN1(pub, digest, rawSig) {
			return errors.New("ECDSA signature verification failed")
		}

	case *rsa.PublicKey:
		if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest, rawSig); err != nil {
			return fmt.Errorf("RSA signature verification failed: %w", err)
		}

	default:
		return fmt.Errorf("unsupported public key type: %T", cert.PublicKey)
	}

	return nil
}

// IsSigned returns true when the Signature carries both a signature and a certificate.
func IsSigned(sig *signv1.Signature) bool {
	return sig != nil && sig.GetSignature() != "" && sig.GetCertificate() != ""
}

// GetOwnerID extracts the owner_id annotation from an ownership Signature.
func GetOwnerID(sig *signv1.Signature) string {
	return sig.GetAnnotations()[ownerIDAnnotation]
}

// canonicalBytes returns the deterministic byte slice that is signed/verified:
// SHA-256( owner_id + ":" + signed_at ).
func canonicalBytes(ownerID, signedAt string) []byte {
	h := sha256.Sum256([]byte(ownerID + ":" + signedAt))

	return h[:]
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
// *x509.Certificate and its DER bytes.
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
