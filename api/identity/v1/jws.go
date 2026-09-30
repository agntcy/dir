// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
)

const (
	// maxCertificateDERSize bounds a certificate embedded in a claim; identity
	// certificates are a few kilobytes.
	maxCertificateDERSize = 16 << 10

	// RSA moduli below 2048 bits are too weak to sign claims; above 8192 bits
	// verifying a claim costs enough to be a denial-of-service vector.
	minRSAKeyBits = 2048
	maxRSAKeyBits = 8192
)

// jwsAlgorithmFor derives the JWS signature algorithm from pub's own
// concrete type. It is never derived from a JWS's self-declared "alg"
// header, so a claim cannot pick its own verification algorithm. Supports
// the subset of the ai-catalog.io Trust Manifest algorithm allowlist this
// package implements: ES256, ES384, EdDSA, RS256.
func jwsAlgorithmFor(pub crypto.PublicKey) (jwa.SignatureAlgorithm, error) {
	switch p := pub.(type) {
	case *ecdsa.PublicKey:
		switch p.Curve {
		case elliptic.P256():
			return jwa.ES256, nil
		case elliptic.P384():
			return jwa.ES384, nil
		default:
			return "", fmt.Errorf("unsupported ECDSA curve: %s", p.Curve.Params().Name)
		}
	case *rsa.PublicKey:
		if bits := p.N.BitLen(); bits < minRSAKeyBits || bits > maxRSAKeyBits {
			return "", fmt.Errorf("unsupported RSA key size: %d bits; %d to %d bits are accepted", bits, minRSAKeyBits, maxRSAKeyBits)
		}

		return jwa.RS256, nil
	case ed25519.PublicKey:
		return jwa.EdDSA, nil
	default:
		return "", fmt.Errorf("unsupported public key type: %T", pub)
	}
}

// CheckSigningKey reports whether claims can be signed and verified with pub:
// ECDSA P-256 or P-384, RSA of 2048 to 8192 bits, or Ed25519.
func CheckSigningKey(pub crypto.PublicKey) error {
	_, err := jwsAlgorithmFor(pub)

	return err
}

// signJWS produces a detached-payload JWS (RFC 7515) compact serialization
// over payload, using signer's own public key type to select the algorithm.
func signJWS(signer crypto.Signer, payload []byte) (string, error) {
	alg, err := jwsAlgorithmFor(signer.Public())
	if err != nil {
		return "", err
	}

	sig, err := jws.Sign(nil, jws.WithKey(alg, signer), jws.WithDetachedPayload(payload))
	if err != nil {
		return "", fmt.Errorf("sign jws: %w", err)
	}

	return string(sig), nil
}

// VerifyJWS verifies a detached-payload JWS compact serialization signature
// against pub over payload. The algorithm is derived from pub's own type
// (see jwsAlgorithmFor), not from the JWS's self-declared "alg" header.
func VerifyJWS(jwsCompact string, pub crypto.PublicKey, payload []byte) error {
	alg, err := jwsAlgorithmFor(pub)
	if err != nil {
		return err
	}

	if _, err := jws.Verify([]byte(jwsCompact), jws.WithKey(alg, pub), jws.WithDetachedPayload(payload)); err != nil {
		return fmt.Errorf("jws verification failed: %w", err)
	}

	return nil
}
