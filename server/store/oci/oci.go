// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"
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
func (s *store) Push(ctx context.Context, mediaType string, rd io.ReadCloser) (*storev1.ObjectRef, error) {
	logger.Debug("Pushing object to OCI store", "mediaType", mediaType)

	// Close reader when done
	defer rd.Close()

	// Step 1: Use oras.PushBytes to push the object data and get Layer Descriptor
	desc, err := s.pushOrSkip(ctx, rd, mediaType)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push object bytes: %v", err)
	}

	// Convert digest to CID
	cid, err := corev1.ConvertDigestToCID(desc.Digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert digest %s to CID: %v", desc.Digest.String(), err)
	}

	// Return object reference
	return &storev1.ObjectRef{Cid: cid}, nil
}

// Lookup checks if the ref exists as a tagged object.
func (s *store) Lookup(ctx context.Context, ref *storev1.ObjectRef) (*storev1.ObjectMeta, error) {
	// Convert ref to digest
	digets, err := corev1.ConvertCIDToDigest(ref.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid CID %s: %v", ref.GetCid(), err)
	}

	// Resolve object
	desc, err := s.repo.Resolve(ctx, digets.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to resolve object with CID %s: %v", ref.GetCid(), err)
	}

	// Extract meta based on media type
	switch desc.MediaType {
	case ocispec.MediaTypeImageManifest:
		// extract meta from manifest
		manifest, manifestDesc, err := s.fetchAndParseManifest(ctx, digets.String())
		if err != nil {
			return nil, err // Error already has proper context from helper
		}

		return &storev1.ObjectMeta{
			Cid:          ref.GetCid(),
			Size:         uint64(manifestDesc.Size),
			MediaType:    manifest.MediaType,
			ArtifactType: manifest.ArtifactType,
			Annotations:  manifest.Annotations,
		}, nil
	case ocispec.MediaTypeImageIndex:
		// extract meta from index
	}

	// any other media type, return only basic info
	return &storev1.ObjectMeta{
		Cid:          ref.GetCid(),
		Size:         uint64(desc.Size),
		MediaType:    desc.MediaType,
		ArtifactType: desc.ArtifactType,
		Annotations:  desc.Annotations,
	}, nil
}

func (s *store) Pull(ctx context.Context, ref *storev1.ObjectRef) (*storev1.ObjectMeta, io.ReadCloser, error) {
	// Fetch object meta
	meta, err := s.Lookup(ctx, ref)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to lookup object with CID %s: %v", ref.GetCid(), err)
	}

	// Convert ref to digest
	digets, err := corev1.ConvertCIDToDigest(ref.GetCid())
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid CID %s: %v", ref.GetCid(), err)
	}

	// Pull object data
	rd, err := s.repo.Fetch(ctx, ocispec.Descriptor{
		Digest: digets,
	})
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to pull object data for CID %s: %v", ref.GetCid(), err)
	}

	return meta, rd, nil
}

func (s *store) Delete(ctx context.Context, ref *storev1.ObjectRef) error {
	logger.Debug("Deleting object from OCI store", "ref", ref)

	panic("unimplemented")
}

// Walk implements types.StoreAPI.
func (s *store) Walk(ctx context.Context, head *storev1.ObjectRef, walkFn func(*storev1.ObjectMeta) error, walkOpts ...func()) error {
	return fmt.Errorf("unimplemented")
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
