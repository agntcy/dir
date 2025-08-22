// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck,nilerr,gosec
package oci

import (
	"context"
	"fmt"
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
)

// ReferrersLister interface for repositories that support the OCI Referrers API.
type ReferrersLister interface {
	Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error
}

// PushSignature stores OCI signature artifacts for a record using cosign attach signature and uploads public key to zot for verification.
func (s *store) PushSignature(ctx context.Context, recordCID string, signature *signv1.Signature) error {
	referrersLogger.Debug("Pushing signature artifact to OCI store", "recordCID", recordCID)

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required")
	}

	// Use cosign attach signature to attach the signature to the record
	err := s.attachSignatureWithCosign(ctx, recordCID, signature)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to attach signature with cosign: %v", err)
	}

	referrersLogger.Debug("Signature attached successfully using cosign", "recordCID", recordCID)

	return nil
}

// PullSignature pulls a signature from the OCI store.
func (s *store) PullSignature(_ context.Context, recordCID string) (*signv1.Signature, error) {
	referrersLogger.Debug("Pulling signature from OCI store", "recordCID", recordCID)

	// TODO implement

	return nil, nil //nolint:nilnil
}

// PushPublicKey pushes a public key as an OCI artifact that references a record as its subject.
func (s *store) PushPublicKey(ctx context.Context, recordCID string, publicKey string) error {
	referrersLogger.Debug("Pushing public key to OCI store", "recordCID", recordCID)

	if len(publicKey) == 0 {
		return status.Error(codes.InvalidArgument, "public key is required")
	}

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required")
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

// PullPublicKey retrieves a public key for a given record CID by finding the public key artifact that references the record.
func (s *store) PullPublicKey(_ context.Context, recordCID string) (string, error) {
	referrersLogger.Debug("Pulling public key from OCI store", "recordCID", recordCID)

	// TODO implement

	return "", nil
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
		return false, err
	}

	// Return the trusted status (which implies signed as well)
	return result.IsTrusted, nil
}
