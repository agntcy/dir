// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
)

var internalLogger = logging.Logger("store/oci/internal")

// pushData pushes record data to OCI and returns the blob descriptor
func (s *store) pushData(ctx context.Context, record *corev1.Record) (ocispec.Descriptor, error) {
	recordCID := record.GetCid()
	if recordCID == "" {
		return ocispec.Descriptor{}, status.Error(codes.InvalidArgument, "record CID is required")
	}

	// Marshal the record using canonical JSON marshaling
	// This ensures the stored content matches the CID calculation
	recordBytes, err := record.CanonicalMarshal()
	if err != nil {
		return ocispec.Descriptor{}, status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	// Calculate digest and size for the descriptor
	// OCI descriptors require these fields to be valid
	ociDigest, err := getDigestFromCID(recordCID)
	if err != nil {
		return ocispec.Descriptor{}, status.Errorf(codes.InvalidArgument, "failed to get digest from CID: %v", err)
	}

	// Create complete blob descriptor
	blobDesc := ocispec.Descriptor{
		MediaType:   "application/json",
		Digest:      ociDigest,
		Size:        int64(len(recordBytes)),
		Annotations: createDescriptorAnnotations(record),
	}

	internalLogger.Debug("Pushing blob to OCI store", "cid", recordCID, "size", len(recordBytes), "mediaType", "application/json")

	// Push JSON blob data - OCI repository will calculate digest automatically
	err = s.repo.Push(ctx, blobDesc, bytes.NewReader(recordBytes))
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return ocispec.Descriptor{}, status.Errorf(codes.Internal, "failed to push blob: %v", err)
	}

	internalLogger.Debug("Blob pushed successfully", "cid", recordCID, "digest", blobDesc.Digest)

	return blobDesc, nil
}

// findAllTagsForRecord discovers all tags that point to a record's manifest
func (s *store) findAllTagsForRecord(ctx context.Context, cid string) ([]string, error) {
	// Both local and remote stores have the same limitation:
	// OCI registries typically don't provide reverse lookup (manifest -> tags)
	// So we reconstruct tags based on record metadata for both cases
	return s.reconstructTagsFromStoredMetadata(ctx, cid)
}

// reconstructTagsFromStoredMetadata rebuilds discovery tags from stored metadata in the registry
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
	return reconstructTagsFromRecord(recordMeta.Annotations, cid), nil
}

// cleanupAllTags removes all discovery tags for a record
func (s *store) cleanupAllTags(ctx context.Context, cid string) error {
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

	return nil // Don't fail delete operation due to tag cleanup issues
}

// deleteFromOCIStore handles deletion of records from an OCI store.
func (s *store) deleteFromOCIStore(ctx context.Context, ref *corev1.RecordRef) error {
	cid := ref.GetCid()
	store := s.repo.(*oci.Store)

	// ENHANCED: Clean up all discovery tags first
	if err := s.cleanupAllTags(ctx, cid); err != nil {
		internalLogger.Debug("Tag cleanup had issues", "cid", cid, "error", err)
		// Continue with deletion even if tag cleanup failed
	}

	// Resolve and delete the manifest. Errors are logged but not returned
	// for the same reason as above.
	manifestDesc, err := s.repo.Resolve(ctx, cid)
	if err != nil {
		internalLogger.Debug("Failed to resolve manifest", "cid", cid, "error", err)
	} else if err := store.Delete(ctx, manifestDesc); err != nil {
		internalLogger.Debug("Failed to delete manifest", "cid", cid, "error", err)
	}

	// Delete the blob associated with the descriptor.
	ociDigest, err := getDigestFromCID(cid)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get digest from CID: %v", err)
	}
	blobDesc := ocispec.Descriptor{
		Digest: ociDigest,
	}
	if err := store.Delete(ctx, blobDesc); err != nil {
		return status.Errorf(codes.Internal, "failed to delete blob: %v", err)
	}

	internalLogger.Info("Record deleted successfully from OCI store", "cid", cid)
	return nil
}

// deleteFromRemoteRepository handles deletion of records from a remote repository.
func (s *store) deleteFromRemoteRepository(ctx context.Context, ref *corev1.RecordRef) error {
	cid := ref.GetCid()
	repo := s.repo.(*remote.Repository)

	// ENHANCED: Clean up all discovery tags first
	if err := s.cleanupAllTags(ctx, cid); err != nil {
		internalLogger.Debug("Tag cleanup had issues", "cid", cid, "error", err)
		// Continue with deletion even if tag cleanup failed
	}

	// Resolve and delete the manifest. Errors are logged but not returned because
	// the record may exist without being tagged with a manifest.
	manifestDesc, err := s.repo.Resolve(ctx, cid)
	if err != nil {
		internalLogger.Debug("Failed to resolve manifest", "cid", cid, "error", err)
	} else if err := repo.Manifests().Delete(ctx, manifestDesc); err != nil {
		internalLogger.Debug("Failed to delete manifest", "cid", cid, "error", err)
	}

	// Delete the blob associated with the descriptor.
	ociDigest, err := getDigestFromCID(cid)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get digest from CID: %v", err)
	}
	blobDesc := ocispec.Descriptor{
		Digest: ociDigest,
	}
	if err := repo.Blobs().Delete(ctx, blobDesc); err != nil {
		return status.Errorf(codes.Internal, "failed to delete blob: %v", err)
	}

	internalLogger.Info("Record deleted successfully from remote repository", "cid", cid)
	return nil
}
