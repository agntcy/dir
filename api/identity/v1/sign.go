// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"
)

// Signer produces a cryptographic proof over a claim's canonical payload.
// It never sees or needs to know about api/sign/v1 — signature production is
// self-contained here to keep this package independent of the sign service.
type Signer interface {
	// Sign returns a detached-payload JWS (RFC 7515) compact serialization
	// over payload.
	Sign(payload []byte) (jwsCompact string, err error)
}

// subjectChecker is optionally implemented by signers that can validate a
// claimed subject against their own credentials before signing.
type subjectChecker interface {
	SubjectMatchesCertificate(subject string) bool
}

// certificateProvider is optionally implemented by signers whose certificate
// must be embedded in the claim so a verifier can validate it without an
// external lookup (SPIFFE).
type certificateProvider interface {
	CertificateDER() []byte
}

// KeySigner signs with a plain PEM-encoded private key (EC, RSA, or
// Ed25519). Used for dns/https/did subjects, whose public key is resolved
// externally by a verifier — the key is never embedded in the claim.
type KeySigner struct {
	signer crypto.Signer
}

// NewKeySigner loads a PEM-encoded private key (EC PRIVATE KEY, RSA PRIVATE
// KEY, or PKCS8 PRIVATE KEY — the PKCS8 form also covers Ed25519).
func NewKeySigner(keyPEM []byte) (*KeySigner, error) {
	signer, err := parsePrivateKey(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &KeySigner{signer: signer}, nil
}

// NewKeySignerFromFile loads a PEM-encoded private key from disk.
func NewKeySignerFromFile(path string) (*KeySigner, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	return NewKeySigner(data)
}

func (s *KeySigner) Sign(payload []byte) (string, error) {
	return signJWS(s.signer, payload)
}

// SpiffeSigner signs with a SPIFFE X.509-SVID key+certificate pair. The
// certificate is embedded in the claim (base64 DER) so a verifier can
// validate the chain and extract the public key without an external lookup.
type SpiffeSigner struct {
	signer   crypto.Signer
	certDER  []byte
	spiffeID string
}

// NewSpiffeSigner loads a PEM-encoded private key and X.509-SVID certificate.
// The certificate must carry a "spiffe://" URI SAN.
func NewSpiffeSigner(keyPEM, certPEM []byte) (*SpiffeSigner, error) {
	signer, err := parsePrivateKey(keyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	cert, certDER, err := parseCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	spiffeID, err := certSpiffeID(cert)
	if err != nil {
		return nil, err
	}

	return &SpiffeSigner{signer: signer, certDER: certDER, spiffeID: spiffeID}, nil
}

// NewSpiffeSignerFromFile loads a PEM-encoded private key and certificate from disk.
func NewSpiffeSignerFromFile(keyPath, certPath string) (*SpiffeSigner, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert file: %w", err)
	}

	return NewSpiffeSigner(keyPEM, certPEM)
}

func (s *SpiffeSigner) Sign(payload []byte) (string, error) {
	return signJWS(s.signer, payload)
}

// CertificateDER implements certificateProvider.
func (s *SpiffeSigner) CertificateDER() []byte {
	return s.certDER
}

var _ certificateProvider = (*SpiffeSigner)(nil)

// SubjectMatchesCertificate reports whether the signer's certificate's
// spiffe:// URI SAN equals subject.
func (s *SpiffeSigner) SubjectMatchesCertificate(subject string) bool {
	return s.spiffeID == subject
}

var _ subjectChecker = (*SpiffeSigner)(nil)

// SignIdentityClaim signs c for recordCID using signer, setting SignedAt,
// Signature, and (for SPIFFE signers) Certificate.
func SignIdentityClaim(c *IdentityClaim, recordCID string, signer Signer) error {
	if c == nil {
		return errors.New("claim is nil")
	}

	signedAt, jwsCompact, certificateB64, err := sign(c.GetSubject(), recordCID, signer)
	if err != nil {
		return err
	}

	c.SignedAt = signedAt
	c.Signature = jwsCompact

	if certificateB64 != "" {
		c.Certificate = strPtr(certificateB64)
	}

	return nil
}

// SignOwnershipClaim signs c for recordCID using signer, setting SignedAt,
// Signature, and (for SPIFFE signers) Certificate.
func SignOwnershipClaim(c *OwnershipClaim, recordCID string, signer Signer) error {
	if c == nil {
		return errors.New("claim is nil")
	}

	signedAt, jwsCompact, certificateB64, err := sign(c.GetSubject(), recordCID, signer)
	if err != nil {
		return err
	}

	c.SignedAt = signedAt
	c.Signature = jwsCompact

	if certificateB64 != "" {
		c.Certificate = strPtr(certificateB64)
	}

	return nil
}

// sign is the shared implementation behind SignIdentityClaim/SignOwnershipClaim.
func sign(subject, recordCID string, signer Signer) (signedAt, jwsCompact, certificateB64 string, err error) {
	if subject == "" {
		return "", "", "", errors.New("subject is required")
	}

	if signer == nil {
		return "", "", "", errors.New("signer is required")
	}

	if checker, ok := signer.(subjectChecker); ok && !checker.SubjectMatchesCertificate(subject) {
		return "", "", "", fmt.Errorf("signer certificate does not match claimed subject %q", subject)
	}

	signedAt = time.Now().UTC().Format(time.RFC3339)

	jwsCompact, err = signer.Sign(CanonicalBytes(recordCID, subject, signedAt))
	if err != nil {
		return "", "", "", err
	}

	if cp, ok := signer.(certificateProvider); ok {
		certificateB64 = base64.StdEncoding.EncodeToString(cp.CertificateDER())
	}

	return signedAt, jwsCompact, certificateB64, nil
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

// parseCertificate decodes a PEM or raw DER certificate and returns the
// parsed *x509.Certificate and its DER bytes.
func parseCertificate(data []byte) (*x509.Certificate, []byte, error) {
	derBytes := data

	if block, _ := pem.Decode(data); block != nil {
		derBytes = block.Bytes
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse certificate: %w", err)
	}

	return cert, derBytes, nil
}

// certSpiffeID returns the certificate's spiffe:// URI SAN.
func certSpiffeID(cert *x509.Certificate) (string, error) {
	for _, u := range cert.URIs {
		if u.Scheme == "spiffe" {
			return u.String(), nil
		}
	}

	return "", errors.New("certificate contains no spiffe:// URI SAN")
}
