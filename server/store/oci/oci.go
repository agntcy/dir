// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	"github.com/agntcy/dir/utils/zot"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
)

var logger = logging.Logger("store/oci")

const (
	// maxTagRetries is the maximum number of retry attempts for Tag operations.
	maxTagRetries = 3
	// initialRetryDelay is the initial delay before the first retry.
	initialRetryDelay = 50 * time.Millisecond
	// maxRetryDelay is the maximum delay between retries.
	maxRetryDelay = 500 * time.Millisecond
)

type store struct {
	repo   oras.GraphTarget
	config ociconfig.Config
}

// Compile-time interface checks to ensure store implements all capability interfaces.
var (
	_ types.StoreAPI = (*store)(nil)
	// _ types.VerifierStore = (*store)(nil)
)

func New(cfg ociconfig.Config) (types.StoreAPI, error) {
	logger.Debug("Creating OCI store with config", "config", cfg)

	// if local dir used, return client for that local path.
	// allows mounting of data via volumes
	// allows S3 usage for backup store
	if repoPath := cfg.LocalDir; repoPath != "" {
		repo, err := oci.New(repoPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create local repo: %w", err)
		}

		return &store{
			repo:   repo,
			config: cfg,
		}, nil
	}

	repo, err := NewORASRepository(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create remote repo: %w", err)
	}

	// Create store API
	return &store{
		repo:   repo,
		config: cfg,
	}, nil
}

// isNotFoundError checks if an error is a "not found" error from the registry.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	return strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "NOT_FOUND")
}

// Walk implements types.StoreAPI.
func (s *store) Walk(ctx context.Context, head *corev1.RecordRef, walkFn func(*corev1.RecordMeta) error, walkOpts ...func()) error {
	panic("unimplemented")
}

// tagWithRetry attempts to tag a manifest with exponential backoff retry logic.
// This is necessary because under concurrent load, oras.PackManifest may push the manifest
// to the registry, but it might not be immediately available when oras.Tag is called.
func (s *store) tagWithRetry(ctx context.Context, manifestDigest, tag string) error {
	var lastErr error

	delay := initialRetryDelay

	for attempt := 0; attempt <= maxTagRetries; attempt++ {
		if attempt > 0 {
			logger.Debug("Retrying Tag operation",
				"attempt", attempt,
				"max_retries", maxTagRetries,
				"delay", delay,
				"manifest_digest", manifestDigest,
				"tag", tag)

			// Wait before retrying
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during tag retry: %w", ctx.Err())
			case <-time.After(delay):
			}

			// Exponential backoff with cap
			delay *= 2
			if delay > maxRetryDelay {
				delay = maxRetryDelay
			}
		}

		// Attempt to tag the manifest
		_, err := oras.Tag(ctx, s.repo, manifestDigest, tag)
		if err == nil {
			if attempt > 0 {
				logger.Info("Tag operation succeeded after retry",
					"attempt", attempt,
					"manifest_digest", manifestDigest,
					"tag", tag)
			}

			return nil
		}

		lastErr = err

		// Only retry on "not found" errors (transient race condition)
		// For other errors, fail immediately
		if !isNotFoundError(err) {
			logger.Debug("Tag operation failed with non-retryable error",
				"error", err,
				"manifest_digest", manifestDigest,
				"tag", tag)

			return fmt.Errorf("failed to tag manifest: %w", err)
		}

		// Log the retryable error
		logger.Debug("Tag operation failed with retryable error",
			"attempt", attempt,
			"error", err,
			"manifest_digest", manifestDigest,
			"tag", tag)
	}

	// All retries exhausted
	logger.Warn("Tag operation failed after all retries",
		"max_retries", maxTagRetries,
		"last_error", lastErr,
		"manifest_digest", manifestDigest,
		"tag", tag)

	return lastErr
}

// Push record to the OCI registry
//
// This creates a blob, a manifest that points to that blob, and a tagged release for that manifest.
// The tag for the manifest is: <CID of digest>.
// The tag for the blob is needed to link the actual record with its associated metadata.
// Note that metadata can be stored in a different store and only wrap this store.
//
// Ref: https://github.com/oras-project/oras-go/blob/main/docs/Modeling-Artifacts.md
func (s *store) Push(ctx context.Context, record *corev1.RecordMeta, rd io.ReadCloser) (*corev1.RecordRef, error) {
	logger.Debug("Pushing record to OCI store", "record", record)

	// Close reader when done
	defer rd.Close()

	// Read all data
	recordBytes, err := io.ReadAll(rd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read record data: %v", err)
	}

	// Step 1: Use oras.PushBytes to push the record data and get Layer Descriptor
	dataDesc, err := oras.PushBytes(ctx, s.repo, "application/json", recordBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push record bytes: %v", err)
	}

	// Step 2: Create manifest
	manifest, err := s.packageRecord(ctx, record, dataDesc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to package record into manifest: %v", err)
	}

	// Step 4: Convert manifest to bytes
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal manifest to JSON: %v", err)
	}

	// Step 3: Push manifest
	manifestDesc, err := oras.PushBytes(ctx, s.repo, ocispec.MediaTypeImageManifest, manifestBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push manifest: %v", err)
	}

	// Step 4: Create RecordRef to return
	cidTag, err := corev1.ConvertDigestToCID(manifestDesc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert manifest digest to CID: %v", err)
	}

	// Step 6: Tag the manifest with CID tag (with retry logic for race conditions)
	// => resolve manifest to record which can be looked up (lookup)
	// => allows pulling record directly (pull)
	if err := s.tagWithRetry(ctx, manifestDesc.Digest.String(), cidTag); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	// Return record reference
	return &corev1.RecordRef{Cid: cidTag}, nil
}

// Lookup checks if the ref exists as a tagged record.
func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	// Pull manifest
	manifest, manifestDesc, err := s.fetchAndParseManifest(ctx, ref.GetCid())
	if err != nil {
		return nil, err // Error already has proper context from helper
	}

	// Convert manifest back to RecordMeta
	meta, _, err := s.unpackageRecord(ctx, manifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unpackage record from manifest for CID %s: %v", ref.GetCid(), err)
	}

	// Set digest
	meta.Cid, err = corev1.ConvertDigestToCID(manifestDesc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert config digest to CID for CID %s: %v", ref.GetCid(), err)
	}
	// Validate CID matches
	if meta.Cid != ref.GetCid() {
		return nil, status.Errorf(codes.Internal, "mismatched CID after lookup: expected %s, got %s", ref.GetCid(), meta.Cid)
	}

	return meta, nil
}

func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, io.ReadCloser, error) {
	// Pull manifest
	manifest, manifestDesc, err := s.fetchAndParseManifest(ctx, ref.GetCid())
	if err != nil {
		return nil, nil, err // Error already has proper context from helper
	}

	// Convert manifest back to RecordMeta
	meta, dataDesc, err := s.unpackageRecord(ctx, manifest)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to unpackage record from manifest for CID %s: %v", ref.GetCid(), err)
	}

	// Validate CID matches
	meta.Cid, err = corev1.ConvertDigestToCID(manifestDesc.Digest)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to convert data digest to CID for CID %s: %v", ref.GetCid(), err)
	}
	if meta.Cid != ref.GetCid() {
		return nil, nil, status.Errorf(codes.Internal, "mismatched CID after pull: expected %s, got %s", ref.GetCid(), meta.Cid)
	}

	// Pull record data blob
	dataReader, err := s.repo.Fetch(ctx, dataDesc)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to pull record data for CID %s: %v", ref.GetCid(), err)
	}

	return meta, dataReader, nil
}

func (s *store) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	logger.Debug("Deleting record from OCI store", "ref", ref)

	// Input validation using shared helper
	if err := validateRecordRef(ref); err != nil {
		return err
	}

	switch s.repo.(type) {
	case *oci.Store:
		return s.deleteFromOCIStore(ctx, ref)
	case *remote.Repository:
		return s.deleteFromRemoteRepository(ctx, ref)
	default:
		return status.Errorf(codes.FailedPrecondition, "unsupported repo type: %T", s.repo)
	}
}

// IsReady checks if the storage backend is ready to serve traffic.
// For local stores, always returns true.
// For remote OCI registries, checks Zot's /readyz endpoint to verify it's ready.
func (s *store) IsReady(ctx context.Context) bool {
	// Local directory stores are always ready
	if s.config.LocalDir != "" {
		logger.Debug("Store ready: using local directory", "path", s.config.LocalDir)

		return true
	}

	// For remote registries, check connectivity
	_, ok := s.repo.(*remote.Repository)
	if !ok {
		// Not a remote repository (could be wrapped), assume ready
		logger.Debug("Store ready: not a remote repository")

		return true
	}

	// Use the zot utility package to check Zot's readiness
	return zot.CheckReadiness(ctx, s.config.RegistryAddress, s.config.Insecure)
}
