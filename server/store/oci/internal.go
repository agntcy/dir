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

	// Validate manifest size if available
	if manifestDesc.Size > 0 {
		internalLogger.Debug("Manifest size from descriptor", "cid", cid, "size", manifestDesc.Size)
	}

	// Fetch manifest from remote
	manifestRd, err := s.repo.Fetch(ctx, manifestDesc)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to fetch manifest for CID %s: %v", cid, err)
	}
	defer manifestRd.Close()

	// Read manifest data
	manifestData, err := io.ReadAll(manifestRd)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to read manifest data for CID %s: %v", cid, err)
	}

	// Validate manifest size matches descriptor
	if manifestDesc.Size > 0 && int64(len(manifestData)) != manifestDesc.Size {
		internalLogger.Warn("Manifest size mismatch",
			"cid", cid,
			"expected", manifestDesc.Size,
			"actual", len(manifestData))
	}

	// Parse manifest
	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to unmarshal manifest for CID %s: %v", cid, err)
	}

	return &manifest, &manifestDesc, nil
}

// findAllTagsForRecord discovers all tags that point to a record's manifest.
func (s *store) findAllTagsForRecord(ctx context.Context, cid string) ([]string, error) {
	// Both local and remote stores have the same limitation:
	// OCI registries typically don't provide reverse lookup (manifest -> tags)
	// So we reconstruct tags based on record metadata for both cases
	return s.reconstructTagsFromStoredMetadata(ctx, cid)
}

// reconstructTagsFromStoredMetadata rebuilds discovery tags from stored metadata in the registry.
func (s *store) reconstructTagsFromStoredMetadata(ctx context.Context, cid string) ([]string, error) {
	// Get the record metadata from manifest annotations
	recordRef := &corev1.RecordRef{Cid: cid}

	recordMeta, err := s.Lookup(ctx, recordRef)
	if err != nil {
		internalLogger.Debug("Failed to lookup record for tag reconstruction", "cid", cid, "error", err)
		// Return at least the CID tag as fallback
		return []string{cid}, nil
	}

	// Use the shared function from tags.go to ensure perfect synchronization
	// This eliminates all duplication and ensures tags match exactly
	return reconstructTagsFromRecord(recordMeta.GetAnnotations(), cid), nil
}

// cleanupAllTags removes all discovery tags for a record.
func (s *store) cleanupAllTags(ctx context.Context, cid string) {
	// Find all tags that might point to this record
	allTags, err := s.findAllTagsForRecord(ctx, cid)
	if err != nil {
		internalLogger.Debug("Failed to find tags for cleanup", "cid", cid, "error", err)
		// Continue with at least the CID tag
		allTags = []string{cid}
	}

	internalLogger.Debug("Cleaning up discovery tags", "cid", cid, "tags", allTags, "count", len(allTags))

	var cleanupErrors []string

	// Remove all tags based on store type
	switch store := s.repo.(type) {
	case *oci.Store:
		// For local OCI store, use Untag
		for _, tag := range allTags {
			if tag == "" {
				continue
			}

			if err := store.Untag(ctx, tag); err != nil {
				internalLogger.Debug("Failed to untag", "tag", tag, "error", err)
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("untag %s: %v", tag, err))
			} else {
				internalLogger.Debug("Successfully removed tag", "tag", tag)
			}
		}

	case *remote.Repository:
		// For remote repositories, tag deletion is often not supported via standard OCI APIs
		// Many registries require manual cleanup or have registry-specific APIs
		internalLogger.Debug("Tag cleanup not supported for remote repository", "cid", cid, "tags", allTags)

		cleanupErrors = append(cleanupErrors, "remote tag cleanup not supported - manual cleanup may be required")
	}

	// Log cleanup summary
	if len(cleanupErrors) > 0 {
		internalLogger.Warn("Some tags could not be cleaned up", "cid", cid, "errors", cleanupErrors)
	} else {
		internalLogger.Info("All discovery tags cleaned up successfully", "cid", cid, "tag_count", len(allTags))
	}
}

// deleteFromOCIStore handles deletion of records from an OCI store.
func (s *store) deleteFromOCIStore(ctx context.Context, ref *corev1.RecordRef) error {
	cid := ref.GetCid()

	store, ok := s.repo.(*oci.Store)
	if !ok {
		return status.Errorf(codes.Internal, "expected *oci.Store, got %T", s.repo)
	}

	internalLogger.Debug("Starting OCI store deletion", "cid", cid)

	var errors []string

	// Phase 1: Remove tag references to manifests (best effort)
	internalLogger.Debug("Phase 1: Cleaning up discovery tags", "cid", cid)
	s.cleanupAllTags(ctx, cid) // This method already handles errors gracefully

	// Phase 2: Remove manifest references to blobs
	internalLogger.Debug("Phase 2: Deleting manifest", "cid", cid)

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

	// Phase 3: Remove blob data (local store - we have full control)
	internalLogger.Debug("Phase 3: Deleting blob data", "cid", cid)

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

	// Phase 1: Remove tag references to manifests (best effort)
	// Note: Many remote registries don't support tag deletion via standard OCI API
	internalLogger.Debug("Phase 1: Attempting tag cleanup (may not be supported)", "cid", cid)
	s.cleanupAllTags(ctx, cid) // This method already handles errors gracefully and logs warnings

	// Phase 2: Remove manifest references to blobs
	internalLogger.Debug("Phase 2: Deleting manifest", "cid", cid)

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

	// Phase 3: Skip blob deletion for remote registries (best practice)
	// Most remote registries handle blob cleanup via garbage collection
	internalLogger.Debug("Phase 3: Skipping blob deletion (handled by registry GC)", "cid", cid)
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
