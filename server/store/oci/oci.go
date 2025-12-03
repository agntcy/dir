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
	storev1 "github.com/agntcy/dir/api/store/v1"
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

// Push object to the OCI registry
//
// This creates a blob, a manifest that points to that blob, and a tagged release for that manifest.
// The tag for the manifest is: <CID of digest>.
// The tag for the blob is needed to link the actual object with its associated metadata.
// Note that metadata can be stored in a different store and only wrap this store.
//
// Ref: https://github.com/oras-project/oras-go/blob/main/docs/Modeling-Artifacts.md
func (s *store) Push(ctx context.Context, object *storev1.Object, rd io.ReadCloser) (*storev1.ObjectRef, error) {
	logger.Debug("Pushing object to OCI store", "object", object)

	// Close reader when done
	defer rd.Close()

	// Step 1: Use oras.PushBytes to push the object data and get Layer Descriptor
	dataDesc, err := s.pushOrSkip(ctx, rd, "application/json")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push object bytes: %v", err)
	}

	// Step 2: Create manifest
	manifest, err := s.packageObject(ctx, object, dataDesc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to package object into manifest: %v", err)
	}

	// Step 4: Convert manifest to bytes
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to marshal manifest to JSON: %v", err)
	}

	// Step 3: Push manifest
	manifestDesc, err := s.pushOrSkip(ctx, io.NopCloser(strings.NewReader(string(manifestBytes))), ocispec.MediaTypeImageManifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push manifest: %v", err)
	}

	// Step 4: Create RecordRef to return
	cidTag, err := corev1.ConvertDigestToCID(manifestDesc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert manifest digest to CID: %v", err)
	}

	// Tag the manifest with the CID tag
	err = s.tagWithRetry(ctx, manifestDesc.Digest.String(), cidTag)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to tag manifest with CID %s: %v", cidTag, err)
	}

	// Return object reference
	return &storev1.ObjectRef{Cid: cidTag}, nil
}

// Lookup checks if the ref exists as a tagged object.
func (s *store) Lookup(ctx context.Context, ref *storev1.ObjectRef) (*storev1.Object, error) {
	// Convert ref to digest
	digets, err := corev1.ConvertCIDToDigest(ref.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid CID %s: %v", ref.GetCid(), err)
	}

	// Pull manifest
	manifest, _, err := s.fetchAndParseManifest(ctx, digets.String())
	if err != nil {
		return nil, err // Error already has proper context from helper
	}

	// Convert manifest back to RecordMeta
	meta, err := s.unpackageObject(ctx, manifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unpackage object from manifest for CID %s: %v", ref.GetCid(), err)
	}

	return meta, nil
}

func (s *store) Pull(ctx context.Context, ref *storev1.ObjectRef) (*storev1.Object, io.ReadCloser, error) {
	// Convert ref to digest
	digets, err := corev1.ConvertCIDToDigest(ref.GetCid())
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid CID %s: %v", ref.GetCid(), err)
	}

	// Pull manifest
	manifest, _, err := s.fetchAndParseManifest(ctx, digets.String())
	if err != nil {
		return nil, nil, err // Error already has proper context from helper
	}

	// Convert manifest back to RecordMeta
	meta, err := s.unpackageObject(ctx, manifest)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to unpackage object from manifest for CID %s: %v", ref.GetCid(), err)
	}

	// Pull object data blob
	dataReader, err := s.repo.Fetch(ctx, manifest.Config)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to pull object data for CID %s: %v", ref.GetCid(), err)
	}

	return meta, dataReader, nil
}

func (s *store) Delete(ctx context.Context, ref *storev1.ObjectRef) error {
	logger.Debug("Deleting object from OCI store", "ref", ref)

	panic("unimplemented")
}

// Walk implements types.StoreAPI.
func (s *store) Walk(ctx context.Context, head *storev1.ObjectRef, walkFn func(*storev1.Object) error, walkOpts ...func()) error {
	panic("unimplemented")
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
