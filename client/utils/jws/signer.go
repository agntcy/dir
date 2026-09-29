// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package jws

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"

	"github.com/youmark/pkcs8"
)

// Signer produces a detached-payload JWS (RFC 7515) compact serialization
// over payload, using key material supplied at construction time.
type Signer interface {
	Sign(payload []byte) (jwsCompact string, err error)
}

// KeySigner signs with a PEM-encoded private key (EC, RSA, or Ed25519),
// optionally password-protected. Used for subjects whose public key is
// resolved externally by a verifier; the key is never embedded in the claim.
type KeySigner struct {
	signer crypto.Signer
}

// NewKeySigner loads a PEM-encoded private key (EC PRIVATE KEY, RSA
// PRIVATE KEY, PKCS8 PRIVATE KEY, or a password-protected ENCRYPTED PRIVATE
// KEY; the PKCS8 forms also cover Ed25519). password is only used for an
// encrypted key and is ignored otherwise; pass nil for an unencrypted key.
func NewKeySigner(keyPEM, password []byte) (*KeySigner, error) {
	signer, err := parsePrivateKey(keyPEM, password)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	return &KeySigner{signer: signer}, nil
}

// NewKeySignerFromFile loads a PEM-encoded private key from disk, see NewKeySigner.
func NewKeySignerFromFile(path string, password []byte) (*KeySigner, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	return NewKeySigner(data, password)
}

func (s *KeySigner) Sign(payload []byte) (string, error) {
	return Sign(s.signer, payload)
}

// KeyCertSigner signs with a private key and embeds its certificate (base64
// DER) alongside the signature, so a verifier can validate the chain and
// extract the public key without an external lookup. Not scheme-specific:
// usable by any subject whose proof is a certificate rather than a
// published key (SPIFFE X.509-SVIDs today).
type KeyCertSigner struct {
	signer  crypto.Signer
	certDER []byte
}

// NewKeyCertSigner loads a PEM-encoded private key and X.509 certificate,
// see NewKeySigner for the key formats and password handling.
func NewKeyCertSigner(keyPEM, certPEM, password []byte) (*KeyCertSigner, error) {
	signer, err := parsePrivateKey(keyPEM, password)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	_, certDER, err := parseCertificate(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse certificate: %w", err)
	}

	return &KeyCertSigner{signer: signer, certDER: certDER}, nil
}

// NewKeyCertSignerFromFile loads a PEM-encoded private key and certificate
// from disk.
func NewKeyCertSignerFromFile(keyPath, certPath string, password []byte) (*KeyCertSigner, error) {
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert file: %w", err)
	}

	return NewKeyCertSigner(keyPEM, certPEM, password)
}

func (s *KeyCertSigner) Sign(payload []byte) (string, error) {
	return Sign(s.signer, payload)
}

// CertificateDER returns the signer's certificate in DER form.
func (s *KeyCertSigner) CertificateDER() []byte {
	return s.certDER
}

// SubjectMatchesCertificate reports whether any URI SAN in the signer's
// certificate equals subject.
func (s *KeyCertSigner) SubjectMatchesCertificate(subject string) bool {
	cert, err := x509.ParseCertificate(s.certDER)
	if err != nil {
		return false
	}

	for _, u := range cert.URIs {
		if u.String() == subject {
			return true
		}
	}

	return false
}

// parsePrivateKey decodes a PEM block and returns a crypto.Signer. password
// is only consulted for an "ENCRYPTED PRIVATE KEY" (PKCS#8 PBES2) block.
func parsePrivateKey(pemBytes, password []byte) (crypto.Signer, error) {
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

		return asSigner(key)

	case "ENCRYPTED PRIVATE KEY":
		if len(password) == 0 {
			return nil, errors.New("private key is encrypted: a password is required")
		}

		key, err := pkcs8.ParsePKCS8PrivateKey(block.Bytes, password)
		if err != nil {
			return nil, fmt.Errorf("decrypt private key: %w", err)
		}

		return asSigner(key)

	default:
		return nil, fmt.Errorf("unsupported PEM block type: %q", block.Type)
	}
}

func asSigner(key any) (crypto.Signer, error) {
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("PKCS8 key type %T does not implement crypto.Signer", key)
	}

	return signer, nil
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
