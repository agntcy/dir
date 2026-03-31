// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
)

var referrersLogger = logging.Logger("store/oci/referrers")

// ReferrerMatcher defines a function type for matching OCI referrer descriptors.
// It returns true if the descriptor matches the expected referrer type.
type ReferrerMatcher func(ctx context.Context, referrer ocispec.Descriptor) bool

// ReferrersLister interface for repositories that support the OCI Referrers API.
type ReferrersLister interface {
	Referrers(ctx context.Context, desc ocispec.Descriptor, artifactType string, fn func(referrers []ocispec.Descriptor) error) error
}

// PushReferrer pushes a generic RecordReferrer as an OCI artifact that references a record as its subject.
// For signature referrers, it uses cosign to attach the signature.
func (s *store) PushReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	referrersLogger.Debug("Pushing referrer to OCI store", "recordCID", recordCID, "type", referrer.GetType())

	if referrer == nil {
		return nil, status.Error(codes.InvalidArgument, "referrer is required") //nolint:wrapcheck
	}

	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	if referrer.GetType() == "" {
		return nil, status.Error(codes.InvalidArgument, "referrer type is required") //nolint:wrapcheck
	}

	if referrer.GetRecordRef() == nil {
		referrer.RecordRef = &corev1.RecordRef{Cid: recordCID}
	} else if referrer.GetRecordRef().GetCid() != recordCID {
		return nil, status.Error(codes.InvalidArgument, "referrer's record CID must match record CID") //nolint:wrapcheck
	}

	// Check if record exists before pushing referrer
	_, err := s.Lookup(ctx, &corev1.RecordRef{Cid: recordCID})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "record not found for CID %s: %v", recordCID, err)
	}

	// Route based on referrer type
	switch referrer.GetType() {
	case corev1.SignatureReferrerType:
		// TODO: validate signature
		return s.pushReferrer(ctx, recordCID, referrer)

	case corev1.PublicKeyReferrerType:
		// TODO: validate public key
		return s.pushReferrer(ctx, recordCID, referrer)

	default:
		// Store as generic OCI referrer
		return s.pushReferrer(ctx, recordCID, referrer)
	}
}

// pushReferrer pushes a referrer as a generic OCI artifact.
func (s *store) pushReferrer(ctx context.Context, recordCID string, referrer *corev1.RecordReferrer) (*corev1.ReferrerRef, error) {
	// Map API type to internal OCI artifact type
	ociArtifactType := apiToOCIType(referrer.GetType())

	// Marshal the referrer to JSON
	referrerBytes, err := referrer.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal referrer: %v", err)
	}

	// Push the referrer blob using internal OCI artifact type
	blobDesc, err := oras.PushBytes(ctx, s.repo, ociArtifactType, referrerBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to push referrer blob: %w", err)
	}

	referrerCID, err := corev1.ConvertDigestToCID(blobDesc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert digest to CID: %v", err)
	}

	// Resolve the record manifest to get its descriptor for the subject field
	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve record manifest for subject: %w", err)
	}

	// Create annotations for the referrer manifest
	annotations := make(map[string]string)
	annotations["agntcy.dir.referrer.type"] = referrer.GetType()
	annotations[ManifestKeyCid] = referrerCID

	if referrer.GetCreatedAt() != "" {
		annotations["agntcy.dir.referrer.created_at"] = referrer.GetCreatedAt()
	}
	// Add custom annotations from the referrer
	for key, value := range referrer.GetAnnotations() {
		annotations["agntcy.dir.referrer.annotation."+key] = value
	}

	// Create the referrer manifest with proper OCI subject field
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			Subject:             &recordManifestDesc,
			ManifestAnnotations: annotations,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to pack referrer manifest: %w", err)
	}

	// Create CID tag for content-addressable storage
	err = s.tagWithRetry(ctx, manifestDesc.Digest.String(), referrerCID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	referrersLogger.Debug("Referrer pushed successfully", "digest", manifestDesc.Digest.String(), "type", referrer.GetType())

	return &corev1.ReferrerRef{Cid: referrerCID}, nil
}

// WalkReferrers walks through referrers for a given record CID, calling walkFn for each referrer.
// If referrerType is empty, all referrers are walked, otherwise only referrers of the specified type.
func (s *store) WalkReferrers(ctx context.Context, recordCID string, referrerType string, walkFn func(*corev1.RecordReferrer) error) error {
	referrersLogger.Debug("Walking referrers from OCI store", "recordCID", recordCID, "type", referrerType)

	if recordCID == "" {
		return status.Error(codes.InvalidArgument, "record CID is required") //nolint:wrapcheck
	}

	if walkFn == nil {
		return status.Error(codes.InvalidArgument, "walkFn is required") //nolint:wrapcheck
	}

	// Get the record manifest descriptor
	recordManifestDesc, err := s.repo.Resolve(ctx, recordCID)
	if err != nil {
		return status.Errorf(codes.NotFound, "failed to resolve record manifest for CID %s: %v", recordCID, err)
	}

	// Determine the matcher based on referrerType
	var matcher ReferrerMatcher

	if referrerType != "" {
		// Map API type to internal OCI artifact type for matching
		ociArtifactType := apiToOCIType(referrerType)

		matcher = s.MediaTypeReferrerMatcher(ociArtifactType)
	}

	// Try the OCI Referrers API first (available on remote registries)
	referrersLister, ok := s.repo.(ReferrersLister)
	if !ok {
		// Fall back to graph Predecessors for local OCI stores
		return s.walkReferrersViaPredecessors(ctx, recordManifestDesc, recordCID, matcher, walkFn)
	}

	var walkErr error

	err = referrersLister.Referrers(ctx, recordManifestDesc, "", func(referrers []ocispec.Descriptor) error {
		for _, referrerDesc := range referrers {
			// Apply matcher if specified
			if matcher != nil && !matcher(ctx, referrerDesc) {
				continue
			}

			// Extract referrer data from manifest
			referrer, err := s.extractReferrerFromManifest(ctx, referrerDesc, recordCID)
			if err != nil {
				referrersLogger.Error("Failed to extract referrer from manifest", "digest", referrerDesc.Digest.String(), "error", err)

				continue // Skip this referrer but continue with others
			}

			// Call the walk function
			if err := walkFn(referrer); err != nil {
				walkErr = err

				return err // Stop walking on error
			}

			referrersLogger.Debug("Referrer processed successfully", "digest", referrerDesc.Digest.String(), "type", referrer.GetType())
		}

		return nil // Continue with next batch
	})

	if walkErr != nil {
		return walkErr
	}

	if err != nil {
		return status.Errorf(codes.Internal, "failed to walk referrers for manifest %s: %v", recordManifestDesc.Digest.String(), err)
	}

	referrersLogger.Debug("Successfully walked referrers", "recordCID", recordCID, "type", referrerType)

	return nil
}

// walkReferrersViaPredecessors walks referrers using the graph Predecessors API.
func (s *store) walkReferrersViaPredecessors(ctx context.Context, subjectDesc ocispec.Descriptor, recordCID string, matcher ReferrerMatcher, walkFn func(*corev1.RecordReferrer) error) error {
	predecessors, err := s.repo.Predecessors(ctx, subjectDesc)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get predecessors for manifest %s: %v", subjectDesc.Digest.String(), err)
	}

	for _, predDesc := range predecessors {
		if predDesc.MediaType != ocispec.MediaTypeImageManifest {
			continue
		}

		if matcher != nil && !matcher(ctx, predDesc) {
			continue
		}

		referrer, err := s.extractReferrerFromManifest(ctx, predDesc, recordCID)
		if err != nil {
			referrersLogger.Error("Failed to extract referrer from manifest", "digest", predDesc.Digest.String(), "error", err)

			continue
		}

		if err := walkFn(referrer); err != nil {
			return err
		}

		referrersLogger.Debug("Referrer processed successfully", "digest", predDesc.Digest.String(), "type", referrer.GetType())
	}

	referrersLogger.Debug("Successfully walked referrers via predecessors", "recordCID", recordCID)

	return nil
}

// extractReferrerFromManifest extracts the referrer data from a referrer manifest.
func (s *store) extractReferrerFromManifest(ctx context.Context, manifestDesc ocispec.Descriptor, recordCID string) (*corev1.RecordReferrer, error) {
	manifest, err := s.fetchAndParseManifestFromDescriptor(ctx, manifestDesc)
	if err != nil {
		return nil, err // Error already includes proper gRPC status
	}

	if len(manifest.Layers) == 0 {
		return nil, status.Errorf(codes.Internal, "referrer manifest has no layers")
	}

	blobDesc := manifest.Layers[0]

	reader, err := s.repo.Fetch(ctx, blobDesc)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "referrer blob not found for CID %s: %v", recordCID, err)
	}
	defer reader.Close()

	referrerData, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read referrer data for CID %s: %v", recordCID, err)
	}

	referrer := &corev1.RecordReferrer{}

	// Try to unmarshal the referrer from JSON
	if err := protojson.Unmarshal(referrerData, referrer); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal referrer for CID %s: %v", recordCID, err)
	}

	// Map internal OCI artifact type back to Dir API type
	if referrer.GetType() != "" {
		referrer.Type = ociToAPIType(referrer.GetType())
	}

	referrerCID, ok := manifest.Annotations[ManifestKeyCid]
	if ok {
		referrer.ReferrerRef = &corev1.ReferrerRef{Cid: referrerCID}
	}

	return referrer, nil
}

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

func (s *store) DeleteReferrer(
	ctx context.Context,
	recordCID string,
	referrerCID string,
	referrerType string,
) ([]string, error) {
	var err error

	cids := []string{}

	err = s.WalkReferrers(
		ctx,
		recordCID,
		referrerType,
		func(referrer *corev1.RecordReferrer) error {
			cid := referrer.GetReferrerRef().GetCid()
			if referrerCID != "" && referrerCID != cid {
				return nil
			}

			cids = append(cids, cid)

			return nil
		},
	)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	result := []string{}

	for _, cid := range cids {
		switch s.repo.(type) {
		case *oci.Store:
			err = s.deleteFromOCIStore(ctx, cid)
		case *remote.Repository:
			err = s.deleteFromRemoteRepository(ctx, cid)
		default:
			err = status.Errorf(codes.FailedPrecondition, "unsupported repo type: %T", s.repo)
		}

		if err != nil {
			return result, err //nolint:wrapcheck
		}

		result = append(result, cid)
	}

	return result, nil
}
