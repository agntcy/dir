// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"
	"strings"

	signv1 "github.com/agntcy/dir/api/sign/v1"
	"github.com/agntcy/dir/utils/cosign"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/zot"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
)

var referrersLogger = logging.Logger("store/oci/referrers")

const (
	// PublicKeyArtifactMediaType defines the media type for public key blobs.
	PublicKeyArtifactMediaType = "application/vnd.agntcy.dir.publickey.v1+pem"

	// SignatureArtifactType defines the media type for cosign signature layers.
	SignatureArtifactType = "application/vnd.dev.cosign.simplesigning.v1+json"
)

// ReferrerMatcher defines a function type for matching OCI referrer descriptors.
// It returns true if the descriptor matches the expected referrer type.
type ReferrerMatcher func(ctx context.Context, referrer ocispec.Descriptor) bool

// MediaTypeReferrerMatcher creates a ReferrerMatcher that checks for a specific media type.
func (s *store) MediaTypeReferrerMatcher(expectedMediaType string) ReferrerMatcher {
	return func(ctx context.Context, referrer ocispec.Descriptor) bool {
		manifest, err := s.fetchAndParseManifestFromDescriptor(ctx, referrer)
		if err != nil {
			referrersLogger.Debug("Failed to fetch and parse referrer manifest", "digest", referrer.Digest.String(), "error", err)

			return false
		}

		// Check if this manifest contains a layer with the expected media type
		return len(manifest.Layers) > 0 && manifest.Layers[0].MediaType == expectedMediaType
	}
}

// ReferrersLister interface for repositories that support the OCI Referrers API.
type ReferrersLister interface {
	Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error
}

// PushSignature stores OCI signature artifacts for a record using cosign attach signature and uploads public key to zot for verification.
func (s *store) PushSignature(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	referrersLogger.Debug("Pushing signature artifact to OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	// Use cosign attach signature to attach the signature to the record
	err := s.attachSignatureWithCosign(ctx, recordCID, signature)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to attach signature with cosign: %v", err)
	}

	referrersLogger.Debug("Signature attached successfully using cosign", "recordCID", recordCID)

	return nil
}

// PullSignatures pulls all signatures from the OCI store for a given record CID.
func (s *store) PullSignatures(ctx context.Context, recordCID string) ([]*signv1.Signature, error) {
	referrersLogger.Debug("Pulling all signatures from OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	// Get the record manifest descriptor
	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to resolve record manifest for CID %s: %v", recordCID, err)
	}

	signatureManifestDescs, err := s.findReferrersByType(ctx, recordManifestDesc, SignatureArtifactType, s.MediaTypeReferrerMatcher(SignatureArtifactType))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find signatures for record CID %s: %v", recordCID, err)
	}

	if len(signatureManifestDescs) == 0 {
		referrersLogger.Debug("No signatures found", "recordCID", recordCID)

		return []*signv1.Signature{}, nil
	}

	// Extract signature data from each manifest
	signatures := make([]*signv1.Signature, 0, len(signatureManifestDescs))

	for _, desc := range signatureManifestDescs {
		signature, err := s.extractSignatureFromManifest(ctx, desc)
		if err != nil {
			referrersLogger.Error("Failed to extract signature from manifest", "digest", desc.Digest.String(), "error", err)

			continue // Skip this signature but continue with others
		}

		signatures = append(signatures, signature)
	}

	referrersLogger.Debug("Successfully pulled signatures", "recordCID", recordCID, "count", len(signatures))

	return signatures, nil
}

// extractSignatureFromManifest extracts signature data from a cosign signature manifest.
func (s *store) extractSignatureFromManifest(ctx context.Context, manifestDesc ocispec.Descriptor) (*signv1.Signature, error) {
	manifest, err := s.fetchAndParseManifestFromDescriptor(ctx, manifestDesc)
	if err != nil {
		return nil, err // Error already includes proper gRPC status
	}

	if len(manifest.Layers) == 0 {
		return nil, status.Errorf(codes.Internal, "signature manifest has no layers")
	}

	signatureBlobDesc := manifest.Layers[0]

	// Validate layer media type
	if signatureBlobDesc.MediaType != SignatureArtifactType {
		referrersLogger.Warn("Unexpected signature blob media type", "expected", SignatureArtifactType, "actual", signatureBlobDesc.MediaType)
	}

	// Fetch the signature data
	blobReader, err := s.repo.Fetch(ctx, signatureBlobDesc)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "signature blob not found for CID %s: %v", manifestDesc.Digest.String(), err)
	}
	defer blobReader.Close()

	signatureBlobData, err := io.ReadAll(blobReader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read signature data for CID %s: %v", manifestDesc.Digest.String(), err)
	}

	// Extract the signature from the layer annotations
	var signatureValue string

	if manifest.Layers[0].Annotations != nil {
		if sig, exists := manifest.Layers[0].Annotations["dev.cosignproject.cosign/signature"]; exists {
			signatureValue = sig
		}
	}

	if signatureValue == "" {
		return nil, status.Errorf(codes.Internal, "no signature value found in annotations")
	}

	return &signv1.Signature{
		Signature: signatureValue,
		Annotations: map[string]string{
			"payload": string(signatureBlobData),
		},
	}, nil
}

// PushPublicKey pushes a public key as an OCI artifact that references a record as its subject.
func (s *store) PushPublicKey(ctx context.Context, recordCID string, publicKey string) error {
	referrersLogger.Debug("Pushing public key to OCI store", "recordCID", recordCID)

	if len(publicKey) == 0 {
		return status.Error(codes.InvalidArgument, "public key is required") //nolint:wrapcheck
	}

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	// Upload the public key to zot for signature verification
	// This enables zot to mark this signature as "trusted" in verification queries
	uploadOpts := &zot.UploadPublicKeyOptions{
		Config:    s.buildZotConfig(),
		PublicKey: publicKey,
	}

	err := zot.UploadPublicKey(ctx, uploadOpts)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to upload public key to zot for verification: %v", err)
	}

	referrersLogger.Debug("Successfully uploaded public key to zot for verification", "recordCID", recordCID)

	// Push the public key blob
	blobDesc, err := oras.PushBytes(ctx, s.repo, PublicKeyArtifactMediaType, []byte(publicKey))
	if err != nil {
		return fmt.Errorf("failed to push public key blob: %w", err)
	}

	// Resolve the record manifest to get its descriptor for the subject field
	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return fmt.Errorf("failed to resolve record manifest for subject: %w", err)
	}

	// Create the public key manifest with proper OCI subject field
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			Subject: &recordManifestDesc,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to pack public key manifest: %w", err)
	}

	referrersLogger.Debug("Public key pushed successfully", "digest", manifestDesc.Digest.String())

	return nil
}

// PullPublicKeys retrieves all public keys for a given record CID by finding all public key artifacts that reference the record.
func (s *store) PullPublicKeys(ctx context.Context, recordCID string) ([]string, error) {
	referrersLogger.Debug("Pulling all public keys from OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to resolve record manifest for CID %s: %v", recordCID, err)
	}

	publicKeyManifestDescs, err := s.findReferrersByType(ctx, recordManifestDesc, PublicKeyArtifactMediaType, s.MediaTypeReferrerMatcher(PublicKeyArtifactMediaType))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find public keys for record CID %s: %v", recordCID, err)
	}

	if len(publicKeyManifestDescs) == 0 {
		referrersLogger.Debug("No public keys found", "recordCID", recordCID)

		return []string{}, nil
	}

	// Extract public key data from each manifest
	publicKeys := make([]string, 0, len(publicKeyManifestDescs))

	for _, desc := range publicKeyManifestDescs {
		publicKey, err := s.extractPublicKeyFromManifest(ctx, desc, recordCID)
		if err != nil {
			referrersLogger.Error("Failed to extract public key from manifest", "digest", desc.Digest.String(), "error", err)

			continue // Skip this public key but continue with others
		}

		publicKeys = append(publicKeys, publicKey)
	}

	referrersLogger.Debug("Successfully pulled public keys", "recordCID", recordCID, "count", len(publicKeys))

	return publicKeys, nil
}

// findReferrersByType searches for all referrer artifacts of the specified type that reference the given record manifest.
func (s *store) findReferrersByType(ctx context.Context, recordManifestDesc ocispec.Descriptor, referrerType string, matcher ReferrerMatcher) ([]ocispec.Descriptor, error) {
	referrersLister, ok := s.repo.(ReferrersLister)
	if !ok {
		return nil, status.Errorf(codes.Unimplemented, "repository does not support OCI referrers API")
	}

	var foundReferrers []ocispec.Descriptor

	err := referrersLister.Referrers(ctx, recordManifestDesc, "", func(referrers []ocispec.Descriptor) error {
		for _, referrer := range referrers {
			if matcher(ctx, referrer) {
				referrersLogger.Debug("Found matching referrer", "type", referrerType, "digest", referrer.Digest.String(), "mediaType", referrer.MediaType)
				foundReferrers = append(foundReferrers, referrer)
			}
		}

		return nil // Continue searching in next batch
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to query referrers for manifest %s: %v", recordManifestDesc.Digest.String(), err)
	}

	return foundReferrers, nil
}

// extractPublicKeyFromManifest extracts the public key data from a public key manifest.
func (s *store) extractPublicKeyFromManifest(ctx context.Context, manifestDesc ocispec.Descriptor, recordCID string) (string, error) {
	manifest, err := s.fetchAndParseManifestFromDescriptor(ctx, manifestDesc)
	if err != nil {
		return "", err // Error already includes proper gRPC status
	}

	if len(manifest.Layers) == 0 {
		return "", status.Errorf(codes.Internal, "public key manifest has no layers")
	}

	blobDesc := manifest.Layers[0]

	reader, err := s.repo.Fetch(ctx, blobDesc)
	if err != nil {
		return "", status.Errorf(codes.NotFound, "public key blob not found for CID %s: %v", recordCID, err)
	}
	defer reader.Close()

	publicKeyData, err := io.ReadAll(reader)
	if err != nil {
		return "", status.Errorf(codes.Internal, "failed to read public key data for CID %s: %v", recordCID, err)
	}

	return string(publicKeyData), nil
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
		Username:  s.config.AuthConfig.Username,
		Password:  s.config.AuthConfig.Password,
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

// buildZotConfig creates a ZotConfig from the store configuration.
func (s *store) buildZotConfig() *zot.VerifyConfig {
	return &zot.VerifyConfig{
		RegistryAddress: s.config.RegistryAddress,
		RepositoryName:  s.config.RepositoryName,
		Username:        s.config.AuthConfig.Username,
		Password:        s.config.AuthConfig.Password,
		AccessToken:     s.config.AuthConfig.AccessToken,
		Insecure:        s.config.AuthConfig.Insecure,
	}
}
