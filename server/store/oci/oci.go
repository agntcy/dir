// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/oci"
)

var logger = logging.Logger("store/oci")

type store struct {
	repo   oras.GraphTarget
	config ociconfig.Config
}

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

// Push raw data to the OCI registry as a blob
func (s *store) PushData(ctx context.Context, reader io.ReadCloser) (*corev1.ObjectRef, error) {
	// Close reader when done
	defer reader.Close()

	// Read all data from the reader
	objectBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read data: %v", err)
	}

	// Check if blob already exists
	digest := ocidigest.FromBytes(objectBytes)
	exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: digest})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check blob existence: %v", err)
	}

	// Push data as a blob to the OCI registry
	if !exists {
		_, err := oras.PushBytes(ctx, s.repo, "", objectBytes)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to push object bytes: %v", err)
		}
	}

	// Convert Digest to CID
	cid, err := corev1.ConvertDigestToCID(digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert digest to CID: %v", err)
	}

	return &corev1.ObjectRef{Cid: cid}, nil
}

// Push object as a manifest to the OCI registry
func (s *store) Push(ctx context.Context, object *corev1.Object) (*corev1.ObjectRef, error) {
	logger.Debug("Pushing object to OCI store", "object", object)

	// Convert object to manifest
	manifest, err := ObjectToManifest(object)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert object to manifest: %v", err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}
	manifestDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageManifest, manifestJSON)

	// Check if manifest already exists
	exists, err := s.repo.Exists(ctx, manifestDesc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check manifest existence: %v", err)
	}
	if !exists {
		// Push manifest
		err = s.repo.Push(ctx, manifestDesc, bytes.NewReader(manifestJSON))
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to push manifest: %v", err)
		}
	}

	// CID to digest
	_, objectRef, err := corev1.MarshalCannonical(object)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert manifest digest to CID: %v", err)
	}

	// Create tag for the manifest using the object CID
	if _, err := oras.Tag(ctx, s.repo, manifestDesc.Digest.String(), objectRef.GetCid()); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	// Return record reference
	return objectRef, nil
}

// Lookup checks if the ref exists as a tagged record.
func (s *store) Lookup(ctx context.Context, ref *corev1.ObjectRef) (*corev1.Object, error) {
	// If the ref points to an object, lookup the data blob
	digest, err := corev1.ConvertCIDToDigest(ref.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert CID to digest: %v", err)
	}
	exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: digest})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check blob existence: %v", err)
	}
	if exists {
		desc, err := s.repo.Resolve(ctx, digest.String())
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
		}

		return &corev1.Object{
			Cid:         ref.GetCid(),
			Size:        uint64(desc.Size),
			Annotations: desc.Annotations,
		}, nil
	}

	// Otherwise, treat as object ref
	desc, err := s.repo.Resolve(ctx, ref.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
	}

	// If the descriptor is a manifest, we need to fetch the config blob
	if desc.MediaType != ocispec.MediaTypeImageManifest {
		return nil, status.Errorf(codes.InvalidArgument, "descriptor is not a manifest: %s", ref.GetCid())
	}

	// Pull and convert to manifest
	manifestRd, err := s.repo.Fetch(ctx, desc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch manifest for CID %s: %v", ref.GetCid(), err)
	}
	defer manifestRd.Close()

	manifestData, err := io.ReadAll(manifestRd)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read manifest data for CID %s: %v", ref.GetCid(), err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal manifest for CID %s: %v", ref.GetCid(), err)
	}

	// Convert manifest to object
	return ManifestToObject(&manifest)
}

func (s *store) Pull(ctx context.Context, ref *corev1.ObjectRef) (io.ReadCloser, error) {
	// Lookup the object first
	obj, err := s.Lookup(ctx, ref)
	if err != nil {
		return nil, err
	}

	// Get data CID
	dataCID := obj.GetCid()
	if obj.Data != nil && obj.Data.GetCid() != "" {
		dataCID = obj.Data.GetCid()
	}

	// Convert CID to digest
	digest, err := corev1.ConvertCIDToDigest(dataCID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert CID to digest: %v", err)
	}

	// Pull the data
	return s.repo.Fetch(ctx, ocispec.Descriptor{Digest: digest})
}

func (s *store) Delete(ctx context.Context, ref *corev1.ObjectRef) error {
	logger.Debug("Deleting record from OCI store", "ref", ref)

	// // Input validation using shared helper
	// if err := validateRecordRef(ref); err != nil {
	// 	return err
	// }

	// switch s.repo.(type) {
	// case *oci.Store:
	// 	return s.deleteFromOCIStore(ctx, ref)
	// case *remote.Repository:
	// 	return s.deleteFromRemoteRepository(ctx, ref)
	// default:
	// 	return status.Errorf(codes.FailedPrecondition, "unsupported repo type: %T", s.repo)
	// }

	return status.Errorf(codes.Unimplemented, "delete operation is not implemented yet")
}
