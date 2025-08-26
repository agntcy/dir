// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
)

var internalLogger = logging.Logger("store/oci/internal")

// validateRecordRef performs common input validation for record reference operations.
// This eliminates duplication across Lookup, Pull, and Delete methods.
func validateRecordRef(ref *corev1.RecordRef) error {
	if ref == nil {
		return status.Error(codes.InvalidArgument, "record reference cannot be nil") //nolint:wrapcheck
	}

	if ref.GetCid() == "" {
		return status.Error(codes.InvalidArgument, "record CID cannot be empty") //nolint:wrapcheck
	}

	return nil
}

// fetchAndParseManifest is a shared helper function that fetches and parses manifests
// for both Lookup and Pull operations, eliminating code duplication.
func (s *store) fetchAndParseManifest(ctx context.Context, cid string) (*ocispec.Manifest, *ocispec.Descriptor, error) {
	// Resolve manifest from remote tag (this also checks existence and validates CID format)
	manifestDesc, err := s.repo.Resolve(ctx, cid)
	if err != nil {
		internalLogger.Debug("Failed to resolve manifest", "cid", cid, "error", err)

		return nil, nil, status.Errorf(codes.NotFound, "record not found: %s", cid)
	}

	internalLogger.Debug("Manifest resolved successfully", "cid", cid, "digest", manifestDesc.Digest.String())

	manifest, err := s.fetchAndParseManifestFromDescriptor(ctx, manifestDesc)
	if err != nil {
		return nil, nil, err
	}

	return manifest, &manifestDesc, nil
}

// fetchAndParseManifestFromDescriptor fetches and parses a manifest when you already have the descriptor.
func (s *store) fetchAndParseManifestFromDescriptor(ctx context.Context, manifestDesc ocispec.Descriptor) (*ocispec.Manifest, error) {
	// Validate manifest size if available
	if manifestDesc.Size > 0 {
		internalLogger.Debug("Manifest size from descriptor", "cid", manifestDesc.Digest.String(), "size", manifestDesc.Size)
	}

	// Fetch manifest from remote
	manifestRd, err := s.repo.Fetch(ctx, manifestDesc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch manifest for %s: %v", manifestDesc.Digest.String(), err)
	}
	defer manifestRd.Close()

	// Read manifest data
	manifestData, err := io.ReadAll(manifestRd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read manifest data for %s: %v", manifestDesc.Digest.String(), err)
	}

	// Validate manifest size matches descriptor
	if manifestDesc.Size > 0 && int64(len(manifestData)) != manifestDesc.Size {
		internalLogger.Warn("Manifest size mismatch",
			"cid", manifestDesc.Digest.String(),
			"expected", manifestDesc.Size,
			"actual", len(manifestData))
	}

	// Parse manifest
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal manifest for %s: %v", manifestDesc.Digest.String(), err)
	}

	return &manifest, nil
}

// Tag cleanup functions removed - OCI registry garbage collection handles dangling tags after manifest deletion

// deleteFromOCIStore handles deletion of records from an OCI store.
func (s *store) deleteFromOCIStore(ctx context.Context, ref *corev1.RecordRef) error {
	cid := ref.GetCid()

	store, ok := s.repo.(*oci.Store)
	if !ok {
		return status.Errorf(codes.Internal, "expected *oci.Store, got %T", s.repo)
	}

	internalLogger.Debug("Starting OCI store deletion", "cid", cid)

	var errors []string

	// Phase 1: Delete manifest (tags will be cleaned up by OCI GC)
	internalLogger.Debug("Phase 1: Deleting manifest", "cid", cid)

	manifestDesc, err := s.repo.Resolve(ctx, cid)
	if err != nil {
		// Manifest might already be gone - this is not necessarily an error
		internalLogger.Debug("Failed to resolve manifest during delete (may already be deleted)", "cid", cid, "error", err)
		errors = append(errors, fmt.Sprintf("manifest resolve: %v", err))
	} else {
		if err := store.Delete(ctx, manifestDesc); err != nil {
			internalLogger.Warn("Failed to delete manifest", "cid", cid, "error", err)
			errors = append(errors, fmt.Sprintf("manifest delete: %v", err))
		} else {
			internalLogger.Debug("Manifest deleted successfully", "cid", cid, "digest", manifestDesc.Digest.String())
		}
	}

	// Phase 2: Remove blob data (local store - we have full control)
	internalLogger.Debug("Phase 2: Deleting blob data", "cid", cid)

	if err := s.deleteBlobForLocalStore(ctx, cid, store); err != nil {
		internalLogger.Warn("Failed to delete blob", "cid", cid, "error", err)
		errors = append(errors, fmt.Sprintf("blob delete: %v", err))
	}

	// Log summary
	if len(errors) > 0 {
		// For local store, we might want to return an error if critical operations failed
		// But continue with best-effort approach for now
		internalLogger.Warn("Partial delete completed with some errors", "cid", cid, "errors", errors)
	} else {
		internalLogger.Info("Record deleted successfully from OCI store", "cid", cid)
	}

	return nil // Best effort - don't fail on partial cleanup
}

// deleteBlobForLocalStore safely deletes blob data from local OCI store using new CID utility.
func (s *store) deleteBlobForLocalStore(ctx context.Context, cid string, store *oci.Store) error {
	// Convert CID to digest using our new utility function
	ociDigest, err := corev1.ConvertCIDToDigest(cid)
	if err != nil {
		return fmt.Errorf("failed to convert CID to digest: %w", err)
	}

	blobDesc := ocispec.Descriptor{
		Digest: ociDigest,
	}

	if err := store.Delete(ctx, blobDesc); err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	internalLogger.Debug("Blob deleted successfully", "cid", cid, "digest", ociDigest.String())

	return nil
}

// deleteFromRemoteRepository handles deletion of records from a remote repository.
func (s *store) deleteFromRemoteRepository(ctx context.Context, ref *corev1.RecordRef) error {
	cid := ref.GetCid()

	repo, ok := s.repo.(*remote.Repository)
	if !ok {
		return status.Errorf(codes.Internal, "expected *remote.Repository, got %T", s.repo)
	}

	internalLogger.Debug("Starting remote repository deletion", "cid", cid)

	var errors []string

	// Phase 1: Delete manifest (tags will be cleaned up by OCI GC)
	internalLogger.Debug("Phase 1: Deleting manifest", "cid", cid)

	manifestDesc, err := s.repo.Resolve(ctx, cid)
	if err != nil {
		internalLogger.Debug("Failed to resolve manifest during delete (may already be deleted)", "cid", cid, "error", err)
		errors = append(errors, fmt.Sprintf("manifest resolve: %v", err))
	} else {
		if err := repo.Manifests().Delete(ctx, manifestDesc); err != nil {
			internalLogger.Warn("Failed to delete manifest", "cid", cid, "error", err)
			errors = append(errors, fmt.Sprintf("manifest delete: %v", err))
		} else {
			internalLogger.Debug("Manifest deleted successfully", "cid", cid, "digest", manifestDesc.Digest.String())
		}
	}

	// Phase 2: Skip blob deletion for remote registries (best practice)
	// Most remote registries handle blob cleanup via garbage collection
	internalLogger.Debug("Phase 2: Skipping blob deletion (handled by registry GC)", "cid", cid)
	internalLogger.Info("Blob cleanup skipped for remote registry - will be handled by garbage collection",
		"cid", cid,
		"note", "This is the recommended approach for remote registries")

	// Log summary
	if len(errors) > 0 {
		// For remote registries, partial failure is common and expected
		// Many operations may not be supported, but this is normal
		internalLogger.Warn("Partial delete completed with some errors", "cid", cid, "errors", errors)
	} else {
		internalLogger.Info("Record deletion completed successfully", "cid", cid)
	}

	return nil // Best effort - remote registries have limited delete capabilities
}
