// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package signing

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/cosign"
)

// computeSignatureDigest computes a unique digest for a signature to use as cache key.
func computeSignatureDigest(sig *signv1.Signature) string {
	// Hash the signature content to create a unique identifier
	h := sha256.New()
	h.Write([]byte(sig.GetSignature()))
	h.Write([]byte(sig.GetContentBundle()))
	return hex.EncodeToString(h.Sum(nil))
}

// Verify verifies record signatures and returns information about all valid signers.
// The verification process:
// 1. Retrieves all signatures from OCI referrers
// 2. For each signature, checks cache first, then verifies if not cached
// 3. Filters based on verification options (if provided)
// 4. Caches verification results
// 5. Returns all valid signers
func (s *sign) Verify(ctx context.Context, recordCID string, options *signv1.VerifyOptions) (*types.VerifyResult, error) {
	logger.Debug("Verifying signature", "recordCID", recordCID)

	// Generate the expected payload for this record CID
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return nil, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	expectedPayload, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return nil, fmt.Errorf("failed to generate expected payload: %w", err)
	}

	// Retrieve signatures from OCI referrers
	signatures, err := s.pullSignatureReferrers(ctx, recordCID)
	if err != nil {
		return nil, fmt.Errorf("failed to pull signature referrers: %w", err)
	}

	if len(signatures) == 0 {
		logger.Debug("No signatures found in referrers", "recordCID", recordCID)
		return &types.VerifyResult{
			Success:      false,
			ErrorMessage: "no signatures found",
		}, nil
	}

	logger.Debug("Retrieved signatures from referrers", "recordCID", recordCID, "count", len(signatures))

	// Retrieve public keys from OCI referrers (for key-based signatures)
	publicKeys, err := s.pullPublicKeyReferrers(ctx, recordCID)
	if err != nil {
		return nil, fmt.Errorf("failed to pull public key referrers: %w", err)
	}

	logger.Debug("Retrieved public keys from referrers", "recordCID", recordCID, "count", len(publicKeys))

	// Verify each signature and collect signer info
	var signers []*signv1.SignerInfo
	legacyMetadata := make(map[string]string)

	for _, sig := range signatures {
		sigDigest := computeSignatureDigest(sig)

		logger.Info("Processing signature", "hasBundle", sig.GetContentBundle() != "", "bundleLen", len(sig.GetContentBundle()), "sigDigest", sigDigest[:8])

		// Check cache first (only if no specific verification options are provided)
		if options == nil && s.db != nil {
			cached, err := s.db.GetSignatureVerification(recordCID, sigDigest)
			if err == nil && cached != nil {
				// Use cached result
				if cached.GetVerified() {
					signerInfo := s.cachedToSignerInfo(cached)
					if signerInfo != nil {
						signers = append(signers, signerInfo)
						logger.Debug("Using cached verification result", "recordCID", recordCID, "sigDigest", sigDigest[:8])
						continue
					}
				} else {
					// Cached as failed, skip
					logger.Info("Skipping cached failed verification", "recordCID", recordCID, "sigDigest", sigDigest[:8])
					continue
				}
			}
		}

		// Verify signature
		signerInfo, err := s.verifySignature(ctx, sig, expectedPayload, publicKeys, options)
		if err != nil {
			logger.Error("Signature verification failed", "error", err, "hasBundle", sig.GetContentBundle() != "")
			// Cache the failed verification (only if no specific options)
			if options == nil && s.db != nil {
				s.cacheVerificationResult(recordCID, sigDigest, nil, err.Error())
			}
			continue
		}

		if signerInfo != nil {
			signers = append(signers, signerInfo)
			// Cache the successful verification (only if no specific options)
			if options == nil && s.db != nil {
				s.cacheVerificationResult(recordCID, sigDigest, signerInfo, "")
			}
		}
	}

	if len(signers) == 0 {
		logger.Debug("No valid signatures found", "recordCID", recordCID)
		return &types.VerifyResult{
			Success:      false,
			ErrorMessage: "no valid signatures found matching verification criteria",
		}, nil
	}

	// Populate legacy metadata from first signer for backward compatibility
	if len(signers) > 0 {
		signer := signers[0]
		if keyInfo := signer.GetKey(); keyInfo != nil {
			legacyMetadata["provider"] = "key"
			legacyMetadata["public_key"] = keyInfo.GetPublicKey()
		} else if oidcInfo := signer.GetOidc(); oidcInfo != nil {
			legacyMetadata["provider"] = "oidc"
			legacyMetadata["oidc.issuer"] = oidcInfo.GetIssuer()
			legacyMetadata["oidc.identity"] = oidcInfo.GetIdentity()
		}
	}

	logger.Info("Signature verification completed", "recordCID", recordCID, "validSigners", len(signers))

	return &types.VerifyResult{
		Success:        true,
		Signers:        signers,
		LegacyMetadata: legacyMetadata,
	}, nil
}

// verifySignature verifies a single signature and returns signer information.
// It handles both key-based and OIDC-based signatures.
func (s *sign) verifySignature(
	ctx context.Context,
	sig *signv1.Signature,
	expectedPayload []byte,
	publicKeys []string,
	options *signv1.VerifyOptions,
) (*signv1.SignerInfo, error) {
	// Check if this is an OIDC-signed signature (has a bundle)
	if sig.GetContentBundle() != "" {
		return s.verifyOIDCSignature(ctx, sig, expectedPayload, options)
	}

	// Otherwise, try key-based verification
	return s.verifyKeySignature(sig, expectedPayload, publicKeys, options)
}

// verifyOIDCSignature verifies a signature with a Sigstore bundle (OIDC-based).
func (s *sign) verifyOIDCSignature(
	ctx context.Context,
	sig *signv1.Signature,
	expectedPayload []byte,
	options *signv1.VerifyOptions,
) (*signv1.SignerInfo, error) {
	// Decode the bundle from base64
	bundleBytes, err := base64.StdEncoding.DecodeString(sig.GetContentBundle())
	if err != nil {
		return nil, fmt.Errorf("failed to decode bundle: %w", err)
	}

	bundleJSON := string(bundleBytes)

	// Parse the bundle to extract OIDC info
	parsedBundle, err := cosign.ParseBundle(bundleJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse bundle: %w", err)
	}

	// Build verification options based on trust root configuration
	verifyOpts := &cosign.VerifyOIDCOptions{
		BundleJSON:      bundleJSON,
		ExpectedPayload: expectedPayload,
	}

	// Detect staging environment from bundle content
	if strings.Contains(bundleJSON, "sigstage.dev") {
		verifyOpts.UseStaging = true
	}

	if oidcOpts := options.GetOidc(); oidcOpts != nil && oidcOpts.GetTrustRoot() != nil {
		tr := oidcOpts.GetTrustRoot()
		verifyOpts.TrustRoot = &cosign.TrustRootConfig{
			FulcioRootPEM:              tr.GetFulcioRootPem(),
			RekorPublicKeyPEM:          tr.GetRekorPublicKeyPem(),
			TimestampAuthorityRootsPEM: tr.GetTimestampAuthorityRootsPem(),
			CTLogPublicKeysPEM:         tr.GetCtLogPublicKeysPem(),
		}
	}

	// Verify the bundle with full Sigstore verification
	result, err := cosign.VerifySignatureWithOIDC(ctx, verifyOpts)
	if err != nil {
		return nil, fmt.Errorf("OIDC verification failed: %w", err)
	}

	if !result.Verified {
		return nil, fmt.Errorf("bundle signature invalid")
	}

	// Extract OIDC info from the bundle
	if parsedBundle.OIDCInfo == nil {
		return nil, fmt.Errorf("no OIDC info in bundle")
	}

	// If options specify OIDC criteria, check if this signature matches
	if oidcOpts := options.GetOidc(); oidcOpts != nil {
		// Check issuer match (if specified)
		if oidcOpts.GetIssuer() != "" && parsedBundle.OIDCInfo.Issuer != oidcOpts.GetIssuer() {
			return nil, nil // Not a match, but not an error
		}
		// Check identity match (if specified)
		if oidcOpts.GetIdentity() != "" && parsedBundle.OIDCInfo.Identity != oidcOpts.GetIdentity() {
			return nil, nil // Not a match, but not an error
		}
	}

	// If options specify key verification, this OIDC signature doesn't match
	if options.GetKey() != nil {
		return nil, nil
	}

	return &signv1.SignerInfo{
		SignerType: &signv1.SignerInfo_Oidc{
			Oidc: &signv1.OIDCSignerInfo{
				Issuer:   parsedBundle.OIDCInfo.Issuer,
				Identity: parsedBundle.OIDCInfo.Identity,
			},
		},
	}, nil
}

// verifyKeySignature verifies a key-based signature.
func (s *sign) verifyKeySignature(
	sig *signv1.Signature,
	expectedPayload []byte,
	publicKeys []string,
	options *signv1.VerifyOptions,
) (*signv1.SignerInfo, error) {
	signatureB64 := sig.GetSignature()

	// If options specify a specific public key, only use that key
	var keysToCheck []string
	if keyOpts := options.GetKey(); keyOpts != nil {
		keysToCheck = []string{keyOpts.GetPublicKey()}
	} else {
		keysToCheck = publicKeys
	}

	// If options specify OIDC verification, skip key-based signatures
	if options.GetOidc() != nil {
		return nil, nil
	}

	// Try each public key
	for _, pubKey := range keysToCheck {
		verified, _, err := cosign.VerifySignatures(&cosign.VerifySignaturesOptions{
			ExpectedPayload: expectedPayload,
			Signatures:      []string{signatureB64},
			PublicKeys:      []string{pubKey},
		})
		if err != nil {
			logger.Debug("Key verification attempt failed", "error", err)
			continue
		}

		if verified {
			return &signv1.SignerInfo{
				SignerType: &signv1.SignerInfo_Key{
					Key: &signv1.KeySignerInfo{
						PublicKey: pubKey,
					},
				},
			}, nil
		}
	}

	return nil, nil // No matching key found
}

// pullSignatureReferrers retrieves signature referrers for a record from OCI registry.
func (s *sign) pullSignatureReferrers(ctx context.Context, recordCID string) ([]*signv1.Signature, error) {
	signatures := make([]*signv1.Signature, 0)

	err := s.store.WalkReferrers(ctx, recordCID, corev1.SignatureReferrerType, func(referrer *corev1.RecordReferrer) error {
		signature := &signv1.Signature{}
		if err := signature.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to decode signature from referrer", "error", err)

			return nil // Skip this referrer but continue walking
		}

		signatures = append(signatures, signature)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull signature referrers: %w", err)
	}

	return signatures, nil
}

// pullPublicKeyReferrers retrieves public key referrers for a record from OCI registry.
func (s *sign) pullPublicKeyReferrers(ctx context.Context, recordCID string) ([]string, error) {
	publicKeys := make([]string, 0)

	err := s.store.WalkReferrers(ctx, recordCID, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		publicKey := &signv1.PublicKey{}
		if err := publicKey.UnmarshalReferrer(referrer); err != nil {
			logger.Debug("Failed to decode public key from referrer", "error", err)

			return nil // Skip this referrer but continue walking
		}

		if publicKey.GetKey() != "" {
			publicKeys = append(publicKeys, publicKey.GetKey())
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to pull public key referrers: %w", err)
	}

	return publicKeys, nil
}

// cacheVerificationResult stores a verification result in the database cache.
func (s *sign) cacheVerificationResult(recordCID, sigDigest string, signerInfo *signv1.SignerInfo, errMsg string) {
	if s.db == nil {
		return
	}

	input := types.SignatureVerificationInput{
		RecordCID:       recordCID,
		SignatureDigest: sigDigest,
		Verified:        signerInfo != nil,
		Error:           errMsg,
	}

	if signerInfo != nil {
		if keyInfo := signerInfo.GetKey(); keyInfo != nil {
			input.SignerType = "key"
			input.PublicKey = keyInfo.GetPublicKey()
		} else if oidcInfo := signerInfo.GetOidc(); oidcInfo != nil {
			input.SignerType = "oidc"
			input.OIDCIssuer = oidcInfo.GetIssuer()
			input.OIDCIdentity = oidcInfo.GetIdentity()
		}
	}

	if err := s.db.CreateSignatureVerification(input); err != nil {
		logger.Debug("Failed to cache verification result", "error", err, "recordCID", recordCID)
	} else {
		logger.Debug("Cached verification result", "recordCID", recordCID, "sigDigest", sigDigest[:8], "verified", input.Verified)
	}
}

// cachedToSignerInfo converts a cached verification to SignerInfo.
func (s *sign) cachedToSignerInfo(cached types.SignatureVerificationObject) *signv1.SignerInfo {
	if !cached.GetVerified() {
		return nil
	}

	switch cached.GetSignerType() {
	case "key":
		return &signv1.SignerInfo{
			SignerType: &signv1.SignerInfo_Key{
				Key: &signv1.KeySignerInfo{
					PublicKey: cached.GetPublicKey(),
				},
			},
		}
	case "oidc":
		return &signv1.SignerInfo{
			SignerType: &signv1.SignerInfo_Oidc{
				Oidc: &signv1.OIDCSignerInfo{
					Issuer:   cached.GetOIDCIssuer(),
					Identity: cached.GetOIDCIdentity(),
				},
			},
		}
	default:
		return nil
	}
}
