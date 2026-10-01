// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package jws

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/youmark/pkcs8"
)

// Signer produces a detached-payload JWS (RFC 7515) compact serialization
// over payload, using key material supplied at construction time.
type Signer interface {
	Sign(payload []byte) (jwsCompact string, err error)
	// Public returns the public key matching the signing key.
	Public() crypto.PublicKey
}

// KeySigner signs with a PEM-encoded private key (EC, RSA, or Ed25519),
// optionally password-protected.
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

func (s *KeySigner) Sign(payload []byte) (string, error) {
	return Sign(s.signer, payload)
}

func (s *KeySigner) Public() crypto.PublicKey {
	return s.signer.Public()
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
