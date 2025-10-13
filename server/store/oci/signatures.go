// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/agntcy/dir/utils/zot"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

// uploadPublicKey uploads a public key to zot for signature verification.
func (s *store) uploadPublicKey(ctx context.Context, referrer *corev1.RecordReferrer) error {
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

// VerifyWithZot queries zot's verification API to check if a signature is valid.
func (s *store) VerifyWithZot(ctx context.Context, recordCID string) (bool, error) {
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
