// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package wellknown

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/agntcy/dir/server/naming"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// ConvertJWKToPublicKey converts a JWK (JSON Web Key) to our internal PublicKey format.
// It extracts the raw public key bytes in DER format for comparison with signing keys.
func ConvertJWKToPublicKey(key jwk.Key) (*naming.PublicKey, error) {
	if key == nil {
		return nil, errors.New("key is nil")
	}

	// Get the raw public key
	var rawKey any
	if err := key.Raw(&rawKey); err != nil {
		return nil, fmt.Errorf("failed to get raw key: %w", err)
	}

	// Convert to DER format based on key type
	var keyBytes []byte

	var keyType string

	var err error

	switch k := rawKey.(type) {
	case *ecdsa.PublicKey:
		keyBytes, err = x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ECDSA key: %w", err)
		}

		keyType = getECDSAKeyType(k)

	case ed25519.PublicKey:
		keyBytes, err = x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal Ed25519 key: %w", err)
		}

		keyType = "ed25519"

	case *rsa.PublicKey:
		keyBytes, err = x509.MarshalPKIXPublicKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal RSA key: %w", err)
		}

		keyType = "rsa"

	default:
		return nil, fmt.Errorf("unsupported key type: %T", rawKey)
	}

	return &naming.PublicKey{
		ID:   key.KeyID(),
		Type: keyType,
		Key:  keyBytes,
	}, nil
}

// getECDSAKeyType returns the key type string for an ECDSA key based on its curve.
func getECDSAKeyType(key *ecdsa.PublicKey) string {
	if key.Curve == nil {
		return "ecdsa"
	}

	switch key.Curve.Params().Name {
	case "P-256":
		return "ecdsa-p256"
	case "P-384":
		return "ecdsa-p384"
	case "P-521":
		return "ecdsa-p521"
	default:
		return "ecdsa"
	}
}
