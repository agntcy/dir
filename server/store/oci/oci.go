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
	storev1 "github.com/agntcy/dir/api/store/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
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
func (s *store) PushData(ctx context.Context, reader io.ReadCloser) (*storev1.ObjectRef, error) {
	// Safely push data
	digest, err := s.pushOrSkip(ctx, reader, "application/octet-stream")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push data: %v", err)
	}

	// Convert digest to CID
	cid, err := corev1.ConvertDigestToCID(digest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert digest to CID: %v", err)
	}

	// Tag the data with object CID
	if _, err := oras.Tag(ctx, s.repo, digest.String(), cid); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	return &storev1.ObjectRef{Cid: cid}, nil
}

// Push object as a manifest to the OCI registry
func (s *store) Push(ctx context.Context, object *storev1.Object) (*storev1.ObjectRef, error) {
	// Convert object to manifest
	manifest, err := ObjectToManifest(object)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert object to manifest: %v", err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Push manifest
	digest, err := s.pushOrSkip(ctx, io.NopCloser(bytes.NewReader(manifestJSON)), ocispec.MediaTypeImageManifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to push manifest: %v", err)
	}

	// Compute CID for the object
	object.Cid = "" // Clear CID to ensure correct computation
	_, objectCID, err := corev1.MarshalCannonical(object)
	if err != nil {
		return nil, fmt.Errorf("failed to compute object CID: %w", err)
	}

	// Tag the manifest data with object CID
	if _, err := oras.Tag(ctx, s.repo, digest.String(), objectCID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create CID tag: %v", err)
	}

	return &storev1.ObjectRef{Cid: objectCID}, nil
}

func (s *store) Lookup(ctx context.Context, ref *storev1.ObjectRef) (*storev1.Object, error) {
	// Get tag for the requested CID
	desc, err := s.repo.Resolve(ctx, ref.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
	}

	// If the media type is not image manifest, return as raw object info
	if desc.MediaType != ocispec.MediaTypeImageManifest {
		return &storev1.Object{
			Cid:  ref.GetCid(),
			Size: uint64(desc.Size),
		}, nil
	}

	// Pull manifest
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
	object, err := ManifestToObject(&manifest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert manifest to object for CID %s: %v", ref.GetCid(), err)
	}

	// Compute CID for the object
	{
		object.Cid = "" // Clear CID to ensure correct computation
		_, objectCID, err := corev1.MarshalCannonical(object)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to compute object CID for CID %s: %v", ref.GetCid(), err)
		}
		object.Cid = objectCID // Set computed CID
	}

	// Verify computed CID matches the requested CID
	if object.GetCid() != ref.GetCid() {
		return nil, status.Errorf(codes.Internal, "object CID mismatch: expected %s, got %s", ref.GetCid(), object.GetCid())
	}

	return object, nil
}

func (s *store) Pull(ctx context.Context, ref *storev1.ObjectRef) (io.ReadCloser, error) {
	// Lookup the object first
	// This also verifies the CID
	obj, err := s.Lookup(ctx, ref)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to lookup object for CID %s: %v", ref.GetCid(), err)
	}

	// If its an actual object, return it as JSON
	if obj.Data != nil {
		objJSON, err := json.Marshal(obj)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to marshal object for CID %s: %v", ref.GetCid(), err)
		}

		return io.NopCloser(bytes.NewReader(objJSON)), nil
	}

	// Convert CID to digest
	digest, err := corev1.ConvertCIDToDigest(obj.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert CID to digest: %v", err)
	}

	// Pull the data
	dataRd, err := s.repo.Fetch(ctx, ocispec.Descriptor{Digest: digest})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch data for CID %s: %v", obj.GetCid(), err)
	}

	return dataRd, nil
}

func (s *store) Delete(ctx context.Context, ref *storev1.ObjectRef) error {
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
