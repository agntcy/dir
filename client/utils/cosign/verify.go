// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"regexp"
	"strings"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	sigs "github.com/sigstore/cosign/v3/pkg/signature"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
)

// VerifySignatureWithOIDC verifies a Sigstore bundle using OIDC identity.
// This performs full Sigstore verification including:
// - Certificate chain validation against Fulcio root
// - Transparency log verification (Rekor)
// - Timestamp verification
// - OIDC issuer and identity matching.
func VerifyWithOIDC(payload []byte, req *signv1.VerifyWithOIDC, signature *signv1.Signature) (*signv1.SignerInfo, error) {
	// Get options with defaults applied
	opts := req.GetOptions().GetDefaultOptions()

	// Parse the bundle
	bundle := &bundle.Bundle{}
	if err := bundle.UnmarshalJSON([]byte(signature.GetContentBundle())); err != nil {
		return nil, fmt.Errorf("failed to parse bundle: %w", err)
	}

	// Get trusted material
	trustedMaterial, err := getOIDCTrustedMaterial(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get trusted material: %w", err)
	}

	// Build verification options based on opts
	verifyOpts := getOIDCVerifierOptions(opts)

	// Create the verifier
	verifier, err := verify.NewVerifier(trustedMaterial, verifyOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create verifier: %w", err)
	}

	// Construct identity matchers based on the request
	issuer, issuerRegexp := getValueMatcher(req.GetIssuer())
	subject, subjectRegexp := getValueMatcher(req.GetSubject())

	// Use permissive identity that matches any issuer/identity
	certIdentity, err := verify.NewShortCertificateIdentity(issuer, issuerRegexp, subject, subjectRegexp)
	if err != nil {
		return nil, fmt.Errorf("failed to create permissive certificate identity: %w", err)
	}

	// Create verification policy
	policy := verify.NewPolicy(
		verify.WithArtifact(bytes.NewReader(payload)),
		verify.WithCertificateIdentity(certIdentity),
	)

	// Verify the bundle
	result, err := verifier.Verify(bundle, policy)
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	return &signv1.SignerInfo{
		Type: &signv1.SignerInfo_Oidc{
			Oidc: &signv1.SignerInfoOIDC{
				Issuer:            result.Signature.Certificate.Issuer,
				Subject:           result.Signature.Certificate.SubjectAlternativeName,
				CertificateIssuer: result.Signature.Certificate.CertificateIssuer,
			},
		},
	}, nil
}

// VerifyWithKeys verifies signatures against public keys using cosign.
// It iterates through all combinations of public keys and signatures to find
// a valid match.
// Public keys can be either PEM content or key references (file paths, URLs, KMS URIs).
//
// Returns true with metadata if any signature verifies with any public key.
// Returns false with nil error if no valid combination is found or if
// no signatures/public keys are provided.
func VerifyWithKeys(ctx context.Context, payload []byte, pubKeys []string, signature *signv1.Signature) (*signv1.SignerInfo, error) {
	// Try each public key against each signature
	for _, publicKey := range pubKeys {
		pubKeyPEM, err := verifySignatureWithKey(ctx, publicKey, signature.GetSignature(), payload)
		if err != nil {
			// Log and continue to try next combination
			continue
		}

		// No error, signature verified successfully with this public key
		return &signv1.SignerInfo{
			Type: &signv1.SignerInfo_Key{
				Key: &signv1.SignerInfoKey{
					PublicKey: pubKeyPEM,
					Algorithm: detectKeyAlgorithm(pubKeyPEM),
				},
			},
		}, nil
	}

	// If we reach here, no valid combination was found
	return nil, fmt.Errorf("no valid signature found for the provided public keys")
}

// ResolvePublicKeyToPEM resolves a key reference to PEM-encoded public key content.
// The keyRef can be PEM content, file path, URL, KMS URI, etc. (same as VerifyWithKey).
func ResolvePublicKeyToPEM(ctx context.Context, keyRef string) (string, error) {
	verifier, err := sigs.LoadPublicKeyRaw([]byte(keyRef), crypto.SHA256)
	if err != nil {
		verifier, err = sigs.PublicKeyFromKeyRefWithHashAlgo(ctx, keyRef, crypto.SHA256)
		if err != nil {
			return "", fmt.Errorf("failed to load public key: %w", err)
		}
	}

	pubKey, err := verifier.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	pubKeyPEM, err := cryptoutils.MarshalPublicKeyToPEM(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key to PEM: %w", err)
	}

	return string(pubKeyPEM), nil
}

// PublicKeyPEMsEqual returns true if the two PEM strings represent the same public key.
func PublicKeyPEMsEqual(pem1, pem2 string) bool {
	if pem1 == pem2 {
		return true
	}

	k1, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pem1))
	if err != nil {
		return false
	}

	k2, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(pem2))
	if err != nil {
		return false
	}

	m1, _ := cryptoutils.MarshalPublicKeyToPEM(k1)
	m2, _ := cryptoutils.MarshalPublicKeyToPEM(k2)

	return string(m1) == string(m2)
}

// verifySignatureWithKey verifies a single signature using a public key.
// The publicKey can be either:
// - PEM-encoded public key content
// - A key reference (file path, URL, or KMS URI)
// Returns the PEM-encoded public key content on success.
func verifySignatureWithKey(ctx context.Context, publicKey string, sig string, expectedPayload []byte) (string, error) {
	pubKeyPEM, err := ResolvePublicKeyToPEM(ctx, publicKey)
	if err != nil {
		return "", err
	}

	verifier, err := sigs.LoadPublicKeyRaw([]byte(pubKeyPEM), crypto.SHA256)
	if err != nil {
		return "", fmt.Errorf("failed to load public key: %w", err)
	}

	// Decode base64 signature
	signatureBytes, err := base64.StdEncoding.DecodeString(sig)
	if err != nil {
		// If decoding fails, assume it's already raw bytes
		signatureBytes = []byte(sig)
	}

	// Verify signature against the expected payload
	err = verifier.VerifySignature(bytes.NewReader(signatureBytes), bytes.NewReader(expectedPayload))
	if err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}

	return pubKeyPEM, nil
}

// getOIDCVerifierOptions builds verifier options for OIDC-based verification.
func getOIDCVerifierOptions(opts *signv1.VerifyOptionsOIDC) []verify.VerifierOption {
	var verifyOpts []verify.VerifierOption

	// Configure transparency log verification
	if opts.GetIgnoreTlog() {
		// Skip transparency log verification - use current time instead of log timestamps
		verifyOpts = append(verifyOpts, verify.WithCurrentTime())
	} else {
		// Require transparency log verification
		verifyOpts = append(verifyOpts, verify.WithTransparencyLog(1))
	}

	// Configure SCT verification
	if !opts.GetIgnoreSct() {
		// Require SCT verification (default behavior)
		verifyOpts = append(verifyOpts, verify.WithSignedCertificateTimestamps(1))
	}

	// Configure TSA verification
	if !opts.GetIgnoreTsa() {
		verifyOpts = append(verifyOpts, verify.WithSignedTimestamps(1))
	}

	return verifyOpts
}

// getOIDCTrustedMaterial returns trusted material for OIDC-based verification.
func getOIDCTrustedMaterial(opts *signv1.VerifyOptionsOIDC) (root.TrustedMaterial, error) {
	switch {
	case opts.GetTrustedRootPath() != "":
		// Option 1: Load trusted root from file path (offline mode)
		trustedRoot, err := root.NewTrustedRootFromPath(opts.GetTrustedRootPath())
		if err != nil {
			return nil, fmt.Errorf("failed to load trusted root from path %s: %w", opts.GetTrustedRootPath(), err)
		}

		return trustedRoot, nil

	case opts.GetTufMirrorUrl() != "":
		// Option 2: Fetch from TUF (online mode)
		tufOpts := tuf.DefaultOptions()
		tufOpts.RepositoryBaseURL = opts.GetTufMirrorUrl()

		// If using staging environment, use staging TUF root
		if strings.Contains(opts.GetTufMirrorUrl(), "sigstage") {
			tufOpts.Root = tuf.StagingRoot()
		}

		trustedMaterial, err := root.FetchTrustedRootWithOptions(tufOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch trusted root from TUF: %w", err)
		}

		return trustedMaterial, nil
	}

	return nil, fmt.Errorf("no trusted root source specified")
}

// getValueMatcher returns a tuple of (exact, regex) matchers based on the input value.
func getValueMatcher(value string) (string, string) {
	// If value is empty, use any-match regex
	if value == "" {
		return "", ".*"
	}

	// If not a valid regexp, return exact matcher
	if _, err := regexp.Compile(value); err != nil {
		return value, ""
	}

	// If a valid regexp, return regexp matcher
	return "", value
}

// detectKeyAlgorithm detects the algorithm from a PEM-encoded public key.
// Returns algorithm name like "ECDSA-P256", "Ed25519", "RSA-2048", etc.
//
//nolint:goconst
func detectKeyAlgorithm(publicKeyPEM string) string {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return "unknown"
	}

	pubKey, err := cryptoutils.UnmarshalPEMToPublicKey([]byte(publicKeyPEM))
	if err != nil {
		return "unknown"
	}

	switch key := pubKey.(type) {
	case *ecdsa.PublicKey:
		if key.Curve != nil {
			return fmt.Sprintf("ECDSA-%s", strings.ToUpper(key.Curve.Params().Name))
		}

		return "ECDSA"
	case ed25519.PublicKey:
		return "Ed25519"
	case *rsa.PublicKey:
		//nolint:mnd
		return fmt.Sprintf("RSA-%d", key.Size()*8)
	default:
		return "unknown"
	}
}
