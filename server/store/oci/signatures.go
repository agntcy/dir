// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/zot"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var signaturesLogger = logging.Logger("store/oci/signatures")

// pushSignature stores OCI signature artifacts for a record using cosign attach signature and uploads public key to zot for verification.
func (s *store) pushSignature(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) error {
	referrersLogger.Debug("Pushing signature artifact to OCI store", "recordCID", recordCID)

	// Decode the signature from the referrer
	signature := &signv1.Signature{}
	if err := signature.UnmarshalReferrer(referrer); err != nil {
		return status.Errorf(codes.Internal, "failed to decode signature from referrer: %v", err)
	}

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	// Use cosign attach signature to attach the signature to the record
	if err := s.attachSignatureWithCosign(ctx, recordCID, signature); err != nil {
		return status.Errorf(codes.Internal, "failed to attach signature with cosign: %v", err)
	}

	referrersLogger.Debug("Signature attached successfully using cosign", "recordCID", recordCID)

	return nil
}

// uploadPublicKeyToZot uploads a public key to Zot's cosign extension for signature verification.
// This is only needed for Zot registries - other registries use OCI referrers only.
func (s *store) uploadPublicKeyToZot(ctx context.Context, referrer *corev1.RecordReferrer) error {
	referrersLogger.Debug("Uploading public key to zot for signature verification")

	// Decode the public key from the referrer
	pk := &signv1.PublicKey{}
	if err := pk.UnmarshalReferrer(referrer); err != nil {
		return status.Errorf(codes.Internal, "failed to get public key from referrer: %v", err)
	}

	publicKey := pk.GetKey()
	if publicKey == "" {
		return status.Error(codes.InvalidArgument, "public key is required") //nolint:wrapcheck
	}

	// Upload the public key to zot for signature verification
	// This enables zot to mark this signature as "trusted" in verification queries
	uploadOpts := &zot.UploadPublicKeyOptions{
		Config:    s.buildZotConfig(),
		PublicKey: publicKey,
	}

	if err := zot.UploadPublicKey(ctx, uploadOpts); err != nil {
		return status.Errorf(codes.Internal, "failed to upload public key to zot for verification: %v", err)
	}

	referrersLogger.Debug("Successfully uploaded public key to zot for verification")

	return nil
}

// attachSignatureWithCosign uses cosign attach signature to attach a signature to a record in the OCI registry.
func (s *store) attachSignatureWithCosign(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	referrersLogger.Debug("Attaching signature using cosign attach signature", "recordCID", recordCID)

	// Construct the OCI image reference for the record
	imageRef := s.constructImageReference(recordCID)

	// Prepare options for attaching signature
	attachOpts := &cosign.AttachSignatureOptions{
		ImageRef:  imageRef,
		Signature: signature.GetSignature(),
		Payload:   signature.GetAnnotations()["payload"],
		Username:  s.config.Username,
		Password:  s.config.Password,
	}

	// Attach signature using utility function
	err := cosign.AttachSignature(ctx, attachOpts)
	if err != nil {
		return fmt.Errorf("failed to attach signature: %w", err)
	}

	referrersLogger.Debug("Cosign attach signature completed successfully")

	return nil
}

// constructImageReference builds the OCI image reference for a record CID.
func (s *store) constructImageReference(recordCID string) string {
	// Get the registry and repository from the config
	registry := s.config.RegistryAddress
	repository := s.config.RepositoryName

	// Remove any protocol prefix from registry address for the image reference
	registry = strings.TrimPrefix(registry, "http://")
	registry = strings.TrimPrefix(registry, "https://")

	// Use CID as tag to match the oras.Tag operation in Push method
	return fmt.Sprintf("%s/%s:%s", registry, repository, recordCID)
}

// buildZotConfig creates a ZotConfig from the store configuration.
func (s *store) buildZotConfig() *zot.VerifyConfig {
	return &zot.VerifyConfig{
		RegistryAddress: s.config.RegistryAddress,
		RepositoryName:  s.config.RepositoryName,
		Username:        s.config.Username,
		Password:        s.config.Password,
		AccessToken:     s.config.AccessToken,
		Insecure:        s.config.Insecure,
	}
}

// convertCosignSignatureToReferrer converts cosign signature data to a referrer.
func (s *store) convertCosignSignatureToReferrer(blobDesc ocispec.Descriptor, data []byte) (*corev1.RecordReferrer, error) {
	// Extract the signature from the layer annotations
	var signatureValue string

	if blobDesc.Annotations != nil {
		if sig, exists := blobDesc.Annotations["dev.cosignproject.cosign/signature"]; exists {
			signatureValue = sig
		}
	}

	if signatureValue == "" {
		return nil, status.Errorf(codes.Internal, "no signature value found in annotations")
	}

	signature := &signv1.Signature{
		Signature: signatureValue,
		Annotations: map[string]string{
			"payload": string(data),
		},
	}

	referrer, err := signature.MarshalReferrer()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encode signature to referrer: %v", err)
	}

	return referrer, nil
}

// VerifySignature verifies a record signature using the appropriate method
// based on the configured registry type.
//
// For Zot registries: Uses Zot's GraphQL API as a fast path for verification.
// If Zot verification fails, falls back to standalone verification.
//
// For other registries (GHCR, DockerHub): Uses standalone verification
// which retrieves signatures and public keys from OCI referrers.
func (s *store) VerifySignature(ctx context.Context, recordCID string) (bool, error) {
	switch s.config.GetType() {
	case ociconfig.RegistryTypeZot:
		// Try Zot's GraphQL API as a fast path
		verified, err := s.verifyWithZot(ctx, recordCID)
		if err == nil && verified {
			return true, nil
		}

		// Fall back to standalone verification if Zot verification fails
		// This handles cases where public keys weren't uploaded to Zot's _cosign dir
		if err != nil {
			referrersLogger.Debug("Zot verification failed, falling back to standalone verification",
				"recordCID", recordCID, "error", err)
		}

		return s.verifyWithReferrers(ctx, recordCID)

	case ociconfig.RegistryTypeGHCR, ociconfig.RegistryTypeDockerHub:
		// Use standalone verification for registries without Zot extensions
		return s.verifyWithReferrers(ctx, recordCID)

	default:
		// For unknown registry types, try standalone verification
		return s.verifyWithReferrers(ctx, recordCID)
	}
}

// verifyWithZot queries zot's verification API to check if a signature is valid.
func (s *store) verifyWithZot(ctx context.Context, recordCID string) (bool, error) {
	verifyOpts := &zot.VerificationOptions{
		Config:    s.buildZotConfig(),
		RecordCID: recordCID,
	}

	result, err := zot.Verify(ctx, verifyOpts)
	if err != nil {
		return false, fmt.Errorf("failed to verify with zot: %w", err)
	}

	// Return the trusted status (which implies signed as well)
	return result.IsTrusted, nil
}

// verifyWithReferrers performs signature verification using OCI referrers.
// This works independently of Zot by:
// 1. Retrieving signatures from OCI referrers
// 2. Retrieving public keys from OCI referrers
// 3. Using shared verification logic to find a valid signature
//
// This approach mirrors Zot's VerifyCosignSignature pattern without requiring Zot extensions.
func (s *store) verifyWithReferrers(ctx context.Context, recordCID string) (bool, error) {
	signaturesLogger.Debug("Starting signature verification with referrers", "recordCID", recordCID)

	// Generate the expected payload for this record CID
	digest, err := corev1.ConvertCIDToDigest(recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	expectedPayload, err := cosign.GeneratePayload(digest.String())
	if err != nil {
		return false, fmt.Errorf("failed to generate expected payload: %w", err)
	}

	// Retrieve signatures from OCI referrers
	signatures, err := s.pullSignatureReferrers(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull signature referrers: %w", err)
	}

	if len(signatures) == 0 {
		signaturesLogger.Debug("No signatures found in referrers", "recordCID", recordCID)

		return false, nil
	}

	signaturesLogger.Debug("Retrieved signatures from referrers", "recordCID", recordCID, "count", len(signatures))

	// Retrieve public keys from OCI referrers
	publicKeys, err := s.pullPublicKeyReferrers(ctx, recordCID)
	if err != nil {
		return false, fmt.Errorf("failed to pull public key referrers: %w", err)
	}

	if len(publicKeys) == 0 {
		signaturesLogger.Debug("No public keys found in referrers", "recordCID", recordCID)

		return false, nil
	}

	signaturesLogger.Debug("Retrieved public keys from referrers", "recordCID", recordCID, "count", len(publicKeys))

	// Convert signatures to string slice for shared verification
	sigStrings := make([]string, len(signatures))
	for i, sig := range signatures {
		sigStrings[i] = sig.GetSignature()
	}

	// Use shared verification logic
	verified, err := cosign.VerifySignatures(&cosign.VerifySignaturesOptions{
		ExpectedPayload: expectedPayload,
		Signatures:      sigStrings,
		PublicKeys:      publicKeys,
	})
	if err != nil {
		return false, fmt.Errorf("failed to verify signatures: %w", err)
	}

	if verified {
		signaturesLogger.Info("Signature verified successfully", "recordCID", recordCID)
	} else {
		signaturesLogger.Debug("No valid signature found for any public key", "recordCID", recordCID)
	}

	return verified, nil
}

// pullSignatureReferrers retrieves signature referrers for a record from OCI registry.
func (s *store) pullSignatureReferrers(ctx context.Context, recordCID string) ([]*signv1.Signature, error) {
	signatures := make([]*signv1.Signature, 0)

	err := s.WalkReferrers(ctx, recordCID, corev1.SignatureReferrerType, func(referrer *corev1.RecordReferrer) error {
		signature := &signv1.Signature{}
		if err := signature.UnmarshalReferrer(referrer); err != nil {
			signaturesLogger.Debug("Failed to decode signature from referrer", "error", err)

			return nil // Skip this referrer but continue walking
		}

		signatures = append(signatures, signature)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return signatures, nil
}

// pullPublicKeyReferrers retrieves public key referrers for a record from OCI registry.
func (s *store) pullPublicKeyReferrers(ctx context.Context, recordCID string) ([]string, error) {
	publicKeys := make([]string, 0)

	err := s.WalkReferrers(ctx, recordCID, corev1.PublicKeyReferrerType, func(referrer *corev1.RecordReferrer) error {
		publicKey := &signv1.PublicKey{}
		if err := publicKey.UnmarshalReferrer(referrer); err != nil {
			signaturesLogger.Debug("Failed to decode public key from referrer", "error", err)

			return nil // Skip this referrer but continue walking
		}

		if publicKey.GetKey() != "" {
			publicKeys = append(publicKeys, publicKey.GetKey())
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return publicKeys, nil
}
