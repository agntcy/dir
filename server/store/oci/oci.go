// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/store/cache"
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
	_ types.StoreAPI         = (*store)(nil)
	_ types.ReferrerStoreAPI = (*store)(nil)
	_ types.VerifierStore    = (*store)(nil)
	_ types.FullStore        = (*store)(nil)
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
	store := &store{
		repo:   repo,
		config: cfg,
	}

	// If no cache requested, return.
	// Do not use in memory cache as it can get large.
	if cfg.CacheDir == "" {
		return store, nil
	}

	// Create cache datastore
	cacheDS, err := datastore.New(datastore.WithFsProvider(cfg.CacheDir))
	if err != nil {
		return nil, fmt.Errorf("failed to create cache store: %w", err)
	}

	// Return cached store
	return cache.Wrap(store, cacheDS), nil
}

// isNotFoundError checks if an error is a "not found" error from the registry.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	return strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "NOT_FOUND")
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
func (s *store) Push(ctx context.Context, record *corev1.Record) (*corev1.RecordRef, error) {
	logger.Debug("Pushing record to OCI store", "record", record)

	// Marshal the record using canonical JSON marshaling first
	// This ensures consistent bytes for both CID calculation and storage
	recordBytes, err := record.Marshal()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal record: %v", err)
	}

	// Step 1: Use oras.PushBytes to push the record data and get Layer Descriptor
	layerDesc, err := oras.PushBytes(ctx, s.repo, "application/json", recordBytes)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push record bytes: %v", err)
	}

	// Step 2: Calculate CID from Layer Descriptor's digest using our new utility function
	recordCID, err := corev1.ConvertDigestToCID(layerDesc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert digest to CID: %v", err)
	}

	// Validate consistency: CID from ORAS digest should match CID from record
	expectedCID := record.GetCid()
	if recordCID != expectedCID {
		return nil, status.Errorf(codes.Internal,
			"CID mismatch: OCI digest CID (%s) != Record CID (%s)",
			recordCID, expectedCID)
	}

	logger.Debug("CID validation successful",
		"cid", recordCID,
		"digest", layerDesc.Digest.String(),
		"validation", "ORAS digest CID matches Record CID")

	logger.Debug("Calculated CID from ORAS digest", "cid", recordCID, "digest", layerDesc.Digest.String())

	// Create record reference
	recordRef := &corev1.RecordRef{Cid: recordCID}

	// Check if record already exists
	if _, err := s.Lookup(ctx, recordRef); err == nil {
		logger.Info("Record already exists in OCI store", "cid", recordCID)

		return recordRef, nil
	}

	// Step 3: Construct manifest annotations and add CID to annotations
	manifestAnnotations := extractManifestAnnotations(record)
	// Add the calculated CID to manifest annotations for discovery
	manifestAnnotations[ManifestKeyCid] = recordCID

	// Step 4: Pack manifest (in-memory only)
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: manifestAnnotations,
			Layers: []ocispec.Descriptor{
				layerDesc,
			},
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pack manifest: %v", err)
	}

	// Step 5: Create CID tag for content-addressable storage
	cidTag := recordCID
	logger.Debug("Generated CID tag", "cid", recordCID, "tag", cidTag)

	// Step 6: Tag the manifest with CID tag (with retry logic for race conditions)
	// => resolve manifest to record which can be looked up (lookup)
	// => allows pulling record directly (pull)
	if err := s.tagWithRetry(ctx, manifestDesc.Digest.String(), cidTag); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	logger.Info("Record pushed to OCI store successfully", "cid", recordCID, "tag", cidTag)

	// Return record reference
	return recordRef, nil
}

// Lookup checks if the ref exists as a tagged record.
func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	// Input validation using shared helper
	if err := validateRecordRef(ref); err != nil {
		return nil, err
	}

	logger.Debug("Starting record lookup", "cid", ref.GetCid())

	// Use shared helper to fetch and parse manifest (eliminates code duplication)
	manifest, _, err := s.fetchAndParseManifest(ctx, ref.GetCid())
	if err != nil {
		return nil, err // Error already has proper context from helper
	}

	// Extract and validate record type from manifest metadata
	recordType, ok := manifest.Annotations[manifestDirObjectTypeKey]
	if !ok {
		return nil, status.Errorf(codes.Internal, "record type not found in manifest annotations for CID %s: missing key %s",
			ref.GetCid(), manifestDirObjectTypeKey)
	}

	// Extract comprehensive metadata from manifest annotations using our enhanced parser
	recordMeta := parseManifestAnnotations(manifest.Annotations)

	// Set the CID from the request (this is the primary identifier)
	recordMeta.Cid = ref.GetCid()

	logger.Debug("Record metadata retrieved successfully",
		"cid", ref.GetCid(),
		"type", recordType,
		"annotationCount", len(manifest.Annotations))

	return recordMeta, nil
}

func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	// Input validation using shared helper
	if err := validateRecordRef(ref); err != nil {
		return nil, err
	}

	logger.Debug("Starting record pull", "cid", ref.GetCid())

	// Use shared helper to fetch and parse manifest (eliminates code duplication)
	manifest, manifestDesc, err := s.fetchAndParseManifest(ctx, ref.GetCid())
	if err != nil {
		return nil, err // Error already has proper context from helper
	}

	// Validate manifest has layers
	if len(manifest.Layers) == 0 {
		return nil, status.Errorf(codes.Internal, "manifest has no layers for CID %s", ref.GetCid())
	}

	// Handle multiple layers with warning
	if len(manifest.Layers) > 1 {
		logger.Warn("Manifest has multiple layers, using first layer",
			"cid", ref.GetCid(),
			"layerCount", len(manifest.Layers))
	}

	// Get the blob descriptor from the first layer
	blobDesc := manifest.Layers[0]

	// Validate layer media type
	if blobDesc.MediaType != "application/json" {
		logger.Warn("Unexpected blob media type",
			"cid", ref.GetCid(),
			"expected", "application/json",
			"actual", blobDesc.MediaType)
	}

	logger.Debug("Fetching record blob",
		"cid", ref.GetCid(),
		"blobDigest", blobDesc.Digest.String(),
		"blobSize", blobDesc.Size,
		"mediaType", blobDesc.MediaType)

	// Fetch the record data using the correct blob descriptor from the manifest
	reader, err := s.repo.Fetch(ctx, blobDesc)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "record blob not found for CID %s: %v", ref.GetCid(), err)
	}
	defer reader.Close()

	// Read all data from the reader
	recordData, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read record data for CID %s: %v", ref.GetCid(), err)
	}

	// Validate blob size matches descriptor
	if blobDesc.Size > 0 && int64(len(recordData)) != blobDesc.Size {
		logger.Warn("Blob size mismatch",
			"cid", ref.GetCid(),
			"expected", blobDesc.Size,
			"actual", len(recordData))
	}

	// Unmarshal canonical JSON data back to Record
	record, err := corev1.UnmarshalRecord(recordData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal record for CID %s: %v", ref.GetCid(), err)
	}

	logger.Debug("Record pulled successfully",
		"cid", ref.GetCid(),
		"blobSize", len(recordData),
		"blobDigest", blobDesc.Digest.String(),
		"manifestDigest", manifestDesc.Digest.String())

	return record, nil
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
