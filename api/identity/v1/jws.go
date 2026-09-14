// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/lestrrat-go/jwx/v2/cert"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jws"
)

const (
	// maxCertificateDERSize bounds a certificate carried in a JWS x5c header
	// or embedded in a claim. Identity certificates are a few kilobytes.
	maxCertificateDERSize = 16 << 10

	// maxCompactJWSSize bounds a compact JWS before its header is decoded. A
	// JWS whose x5c carries a maximum-size leaf is about 29 KiB.
	maxCompactJWSSize = 64 << 10

	// compactSegments is the number of dot-separated segments of a compact
	// JWS serialization.
	compactSegments = 3
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
		return jwa.RS256, nil
	case ed25519.PublicKey:
		return jwa.EdDSA, nil
	default:
		return "", fmt.Errorf("unsupported public key type: %T", pub)
	}
}

// CheckSigningKey reports whether claims can be signed and verified with pub:
// ECDSA P-256 or P-384, RSA, or Ed25519. Signers and resolvers share this one
// allowlist.
func CheckSigningKey(pub crypto.PublicKey) error {
	_, err := jwsAlgorithmFor(pub)

	return err
}

// signJWS produces a detached-payload JWS (RFC 7515) compact serialization
// over payload, using signer's own public key type to select the algorithm.
func signJWS(signer crypto.Signer, payload []byte) (string, error) {
	return signJWSWithHeaders(signer, jws.NewHeaders(), payload)
}

// signJWSWithCertificate is signJWS with the certificate, given as the
// standard base64 of its DER, carried in the protected header as x5c
// (RFC 7515 section 4.1.6).
func signJWSWithCertificate(signer crypto.Signer, certB64 string, payload []byte) (string, error) {
	var chain cert.Chain
	if err := chain.AddString(certB64); err != nil {
		return "", fmt.Errorf("x5c: %w", err)
	}

	hdrs := jws.NewHeaders()
	if err := hdrs.Set(jws.X509CertChainKey, &chain); err != nil {
		return "", fmt.Errorf("x5c: %w", err)
	}

	return signJWSWithHeaders(signer, hdrs, payload)
}

// signJWSWithHeaders signs payload with the given protected headers.
// jws.Sign writes the algorithm into hdrs, so callers pass a fresh value.
func signJWSWithHeaders(signer crypto.Signer, hdrs jws.Headers, payload []byte) (string, error) {
	alg, err := jwsAlgorithmFor(signer.Public())
	if err != nil {
		return "", err
	}

	sig, err := jws.Sign(nil, jws.WithKey(alg, signer, jws.WithProtectedHeaders(hdrs)), jws.WithDetachedPayload(payload))
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

// CertificateFromJWS returns the certificate a compact JWS carries in its
// protected x5c header. The certificate is untrusted: neither its chain nor
// its subject is validated here, which departs from RFC 7515 section 4.1.6,
// and the signature has not been checked against its key. The caller decides
// what makes the certificate trustworthy, for instance an attestation of its
// fingerprint by a transparency log, and then verifies the JWS with VerifyJWS.
func CertificateFromJWS(jwsCompact string) (*x509.Certificate, error) {
	if len(jwsCompact) > maxCompactJWSSize {
		return nil, fmt.Errorf("jws is %d bytes; at most %d are accepted", len(jwsCompact), maxCompactJWSSize)
	}

	segments := strings.Split(jwsCompact, ".")
	if len(segments) != compactSegments {
		return nil, fmt.Errorf("jws has %d segments; a compact serialization has %d", len(segments), compactSegments)
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(segments[0])
	if err != nil {
		return nil, fmt.Errorf("decode jws header: %w", err)
	}

	hdrs := jws.NewHeaders()
	if err := json.Unmarshal(headerJSON, hdrs); err != nil {
		return nil, fmt.Errorf("parse jws header: %w", err)
	}

	chain := hdrs.X509CertChain()
	if chain == nil || chain.Len() == 0 {
		return nil, errors.New("jws carries no x5c certificate")
	}

	if chain.Len() != 1 {
		return nil, fmt.Errorf("jws x5c carries %d certificates; the leaf only is accepted", chain.Len())
	}

	encoded, _ := chain.Get(0)
	if len(encoded) > base64.StdEncoding.EncodedLen(maxCertificateDERSize) {
		return nil, fmt.Errorf("x5c certificate exceeds %d bytes", maxCertificateDERSize)
	}

	der, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("x5c certificate is not standard base64 (RFC 7515 section 4.1.6): %w", err)
	}

	if len(der) > maxCertificateDERSize {
		return nil, fmt.Errorf("x5c certificate is %d bytes; at most %d are accepted", len(der), maxCertificateDERSize)
	}

	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("parse x5c certificate: %w", err)
	}

	return parsed, nil
}
