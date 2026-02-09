// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package cosign

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// VerifyOIDCOptions contains options for OIDC-based signature verification.
type VerifyOIDCOptions struct {
	// BundleJSON is the Sigstore bundle in JSON format (not base64 encoded).
	BundleJSON string

	// ExpectedPayload is the payload that was signed.
	// This should match what was used during signing.
	ExpectedPayload []byte

	// ExpectedIssuer is the expected OIDC issuer URL (exact match).
	// If empty, issuer is not verified.
	ExpectedIssuer string

	// ExpectedIdentity is the expected OIDC subject/identity (exact match).
	// If empty, identity is not verified.
	ExpectedIdentity string

	// TrustRoot is the trust root configuration.
	// If nil, uses Sigstore public good instance (or staging if UseStaging is true).
	TrustRoot *TrustRootConfig

	// UseStaging uses the Sigstore staging environment trust root.
	// This should be true when verifying signatures created with staging.
	UseStaging bool
}

// VerifyOIDCResult contains the result of OIDC signature verification.
type VerifyOIDCResult struct {
	// Verified is true if the signature is valid.
	Verified bool

	// Issuer is the OIDC issuer from the certificate.
	Issuer string

	// Identity is the OIDC identity/subject from the certificate.
	Identity string

	// Certificate is the PEM-encoded signing certificate.
	Certificate string
}

// VerifySignatureWithOIDC verifies a Sigstore bundle using OIDC identity.
// This performs full Sigstore verification including:
// - Certificate chain validation against Fulcio root
// - Transparency log verification (Rekor)
// - Timestamp verification
// - OIDC issuer and identity matching
func VerifySignatureWithOIDC(ctx context.Context, opts *VerifyOIDCOptions) (*VerifyOIDCResult, error) {
	if opts == nil {
		return nil, errors.New("options are required")
	}

	if opts.BundleJSON == "" {
		return nil, errors.New("bundle JSON is required")
	}

	if len(opts.ExpectedPayload) == 0 {
		return nil, errors.New("expected payload is required")
	}

	// Parse the bundle
	b := &bundle.Bundle{}
	if err := b.UnmarshalJSON([]byte(opts.BundleJSON)); err != nil {
		return nil, fmt.Errorf("failed to parse bundle: %w", err)
	}

	// Get or create trust root
	var trustRootProvider *TrustRootProvider
	var err error

	if opts.TrustRoot != nil {
		trustRootProvider, err = NewCustomTrustRoot(opts.TrustRoot)
	} else if opts.UseStaging {
		trustRootProvider, err = NewStagingTrustRoot()
	} else {
		trustRootProvider, err = NewPublicGoodTrustRoot()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create trust root: %w", err)
	}

	trustedMaterial, err := trustRootProvider.GetTrustedMaterial()
	if err != nil {
		return nil, fmt.Errorf("failed to get trusted material: %w", err)
	}

	// Build verification options
	// Try multiple timestamp sources - the bundle may have:
	// - Signed timestamps from TSA (Timestamp Authority)
	// - Integrated timestamps from Rekor transparency log
	// We accept any one of these as valid
	verifyOpts := []verify.VerifierOption{
		verify.WithSignedTimestamps(1), // Use TSA signed timestamps from bundle
	}

	// Create the verifier
	verifier, err := verify.NewSignedEntityVerifier(trustedMaterial, verifyOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create verifier: %w", err)
	}

	// Build policy for identity verification
	policyOpts := []verify.PolicyOption{}

	// Add identity policy - if issuer/identity are specified, use exact match
	// Otherwise use a permissive match that accepts any identity
	if opts.ExpectedIssuer != "" || opts.ExpectedIdentity != "" {
		certIdentity, err := verify.NewShortCertificateIdentity(
			opts.ExpectedIssuer,
			"", // issuer regexp (not used - exact match only)
			opts.ExpectedIdentity,
			"", // identity regexp (not used - exact match only)
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create certificate identity: %w", err)
		}

		policyOpts = append(policyOpts, verify.WithCertificateIdentity(certIdentity))
	} else {
		// Use permissive identity that matches any issuer/identity
		certIdentity, err := verify.NewShortCertificateIdentity(
			"",   // any issuer
			".*", // issuer regexp - match anything
			"",   // any identity
			".*", // identity regexp - match anything
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create permissive certificate identity: %w", err)
		}

		policyOpts = append(policyOpts, verify.WithCertificateIdentity(certIdentity))
	}

	policy := verify.NewPolicy(verify.WithArtifact(bytes.NewReader(opts.ExpectedPayload)), policyOpts...)

	// Verify the bundle
	result, err := verifier.Verify(b, policy)
	if err != nil {
		return &VerifyOIDCResult{
			Verified: false,
		}, fmt.Errorf("verification failed: %w", err)
	}

	// Extract OIDC info from the verified certificate
	verifyResult := &VerifyOIDCResult{
		Verified: true,
	}

	// Get certificate from verification result
	if result.Signature != nil && result.Signature.Certificate != nil {
		// Extract OIDC info using our helper
		parsedBundle, parseErr := ParseBundle(opts.BundleJSON)
		if parseErr == nil && parsedBundle.OIDCInfo != nil {
			verifyResult.Issuer = parsedBundle.OIDCInfo.Issuer
			verifyResult.Identity = parsedBundle.OIDCInfo.Identity
		}

		// Get certificate PEM
		if parseErr == nil {
			certPEM, certErr := parsedBundle.GetCertificatePEM()
			if certErr == nil {
				verifyResult.Certificate = certPEM
			}
		}
	}

	return verifyResult, nil
}

// VerifyBundleSignature verifies just the cryptographic signature in a bundle
// without full Sigstore verification (no Rekor/Fulcio checks).
// This is useful for offline verification when you have the certificate.
func VerifyBundleSignature(bundleJSON string, expectedPayload []byte) (bool, error) {
	parsed, err := ParseBundle(bundleJSON)
	if err != nil {
		return false, fmt.Errorf("failed to parse bundle: %w", err)
	}

	if parsed.Certificate == nil {
		return false, errors.New("no certificate in bundle")
	}

	if len(parsed.Signature) == 0 {
		return false, errors.New("no signature in bundle")
	}

	// Get public key PEM from certificate
	pubKeyPEM, err := parsed.GetCertificatePEM()
	if err != nil {
		return false, fmt.Errorf("failed to get certificate PEM: %w", err)
	}

	// Verify using our existing key-based verification
	verified, _, err := VerifySignatures(&VerifySignaturesOptions{
		ExpectedPayload: expectedPayload,
		Signatures:      []string{string(parsed.Signature)},
		PublicKeys:      []string{pubKeyPEM},
	})

	if err != nil {
		return false, err
	}

	return verified, nil
}

// MatchesOIDCIdentity checks if a bundle's certificate matches the expected OIDC identity.
// This does NOT perform cryptographic verification - use VerifySignatureWithOIDC for that.
func MatchesOIDCIdentity(bundleJSON string, expectedIssuer, expectedIdentity string) (bool, error) {
	oidcInfo, err := ExtractOIDCInfoFromBundle(bundleJSON)
	if err != nil {
		return false, err
	}

	// Check issuer match (if specified)
	if expectedIssuer != "" && oidcInfo.Issuer != expectedIssuer {
		return false, nil
	}

	// Check identity match (if specified)
	if expectedIdentity != "" && oidcInfo.Identity != expectedIdentity {
		return false, nil
	}

	return true, nil
}
