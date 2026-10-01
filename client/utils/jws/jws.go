// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package jws signs and verifies detached-payload JWS (RFC 7515) signatures.
// It is payload-agnostic: callers decide what bytes get signed. Like
// client/utils/cosign, it lives outside api/*, which is proto types and
// data-shape glue only, never crypto libraries or I/O.
package jws

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"errors"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	jwxjws "github.com/lestrrat-go/jwx/v2/jws"
)

// minRSAKeyBits is the smallest RSA modulus accepted; Go's own floor is lower.
const minRSAKeyBits = 2048

// jwsAlgorithmFor derives the JWS signature algorithm from pub's own
// concrete type. It is never derived from a JWS's self-declared "alg"
// header, so a claim cannot pick its own verification algorithm. Supports
// ES256, ES384, EdDSA, RS256 (RSA keys of at least minRSAKeyBits).
func jwsAlgorithmFor(pub crypto.PublicKey) (jwa.SignatureAlgorithm, error) {
	switch p := pub.(type) {
	case *ecdsa.PublicKey:
		if p == nil || p.Curve == nil {
			return "", errors.New("malformed ECDSA public key")
		}

		switch p.Curve {
		case elliptic.P256():
			return jwa.ES256, nil
		case elliptic.P384():
			return jwa.ES384, nil
		default:
			return "", fmt.Errorf("unsupported ECDSA curve: %s", p.Curve.Params().Name)
		}
	case *rsa.PublicKey:
		if p == nil || p.N == nil {
			return "", errors.New("malformed RSA public key")
		}

		if bits := p.N.BitLen(); bits < minRSAKeyBits {
			return "", fmt.Errorf("RSA key too small: %d bits, need at least %d", bits, minRSAKeyBits)
		}

		return jwa.RS256, nil
	case ed25519.PublicKey:
		if len(p) != ed25519.PublicKeySize {
			return "", fmt.Errorf("malformed Ed25519 public key: %d bytes", len(p))
		}

		return jwa.EdDSA, nil
	default:
		return "", fmt.Errorf("unsupported public key type: %T", pub)
	}
}

// Sign produces a detached-payload JWS (RFC 7515) compact serialization
// over payload, using signer's own public key type to select the algorithm.
func Sign(signer crypto.Signer, payload []byte) (string, error) {
	alg, err := jwsAlgorithmFor(signer.Public())
	if err != nil {
		return "", err
	}

	sig, err := jwxjws.Sign(nil, jwxjws.WithKey(alg, signer), jwxjws.WithDetachedPayload(payload))
	if err != nil {
		return "", fmt.Errorf("sign jws: %w", err)
	}

	return string(sig), nil
}

// Verify verifies a detached-payload JWS compact serialization signature
// over payload and succeeds if any of keys validates it. Each key's algorithm
// is derived from the key's own type (see jwsAlgorithmFor), not from the
// JWS's self-declared "alg" header. A key that can't be used (unsupported
// type) doesn't stop the remaining keys from being tried.
func Verify(jwsCompact string, payload []byte, keys ...crypto.PublicKey) error {
	if len(keys) == 0 {
		return errors.New("no public keys to verify against")
	}

	errs := make([]error, 0, len(keys))

	for _, key := range keys {
		err := verifyWithKey(jwsCompact, payload, key)
		if err == nil {
			return nil
		}

		errs = append(errs, err)
	}

	return fmt.Errorf("jws verification failed against %d key(s): %w", len(keys), errors.Join(errs...))
}

func verifyWithKey(jwsCompact string, payload []byte, pub crypto.PublicKey) error {
	alg, err := jwsAlgorithmFor(pub)
	if err != nil {
		return err
	}

	if _, err := jwxjws.Verify([]byte(jwsCompact), jwxjws.WithKey(alg, pub), jwxjws.WithDetachedPayload(payload)); err != nil {
		return fmt.Errorf("jws verification failed: %w", err)
	}

	return nil
}
