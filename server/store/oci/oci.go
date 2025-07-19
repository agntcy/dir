// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck,nilerr,gosec
package oci

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	corev1 "github.com/agntcy/dir/api/core/v1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/store/cache"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"

	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

var logger = logging.Logger("store/oci")

type store struct {
	repo oras.GraphTarget
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
			repo: repo,
		}, nil
	}

	// create remote client
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", cfg.RegistryAddress, cfg.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// configure client to remote
	repo.PlainHTTP = cfg.Insecure
	repo.Client = &auth.Client{
		Client: retry.DefaultClient,
		Header: http.Header{
			"User-Agent": {"dir-client"},
		},
		Cache: auth.DefaultCache,
		Credential: auth.StaticCredential(
			cfg.RegistryAddress,
			auth.Credential{
				Username:     cfg.Username,
				Password:     cfg.Password,
				RefreshToken: cfg.RefreshToken,
				AccessToken:  cfg.AccessToken,
			},
		),
	}

	// Create store API
	store := &store{
		repo: repo,
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

	recordCID := record.GetCid()
	if recordCID == "" {
		return nil, status.Error(codes.InvalidArgument, "record CID is required")
	}

	logger.Debug("Using CID from record", "cid", recordCID)

	// Create record reference
	recordRef := &corev1.RecordRef{Cid: recordCID}

	// Check if record already exists
	if _, err := s.Lookup(ctx, recordRef); err == nil {
		logger.Info("Record already exists in OCI store", "cid", recordCID)
		return recordRef, nil
	}

	// Push the record data using the private pushData method
	blobDesc, err := s.pushData(ctx, record)
	if err != nil {
		return nil, err
	}

	// Extract rich manifest annotations for discovery using our helper function
	manifestAnnotations := extractManifestAnnotations(record)

	// Push manifest
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: manifestAnnotations,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to pack manifest: %v", err)
	}

	// Generate multiple discovery tags for enhanced browsability
	discoveryTags := generateDiscoveryTags(record, DefaultTagStrategy)
	logger.Debug("Generated discovery tags", "cid", recordCID, "tags", discoveryTags, "count", len(discoveryTags))

	// Push manifest with multiple tags
	// tags => resolve manifest to record which can be looked up (lookup)
	// tags => allow to pull record directly (pull)
	// tags => allow listing and filtering tags (list)
	err = s.pushManifestWithTags(ctx, manifestDesc, discoveryTags)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create discovery tags: %v", err)
	}

	logger.Info("Record pushed to OCI store successfully", "cid", recordCID, "tags", len(discoveryTags))

	// Return record reference
	return recordRef, nil
}

// Lookup checks if the ref exists as a tagged record.
func (s *store) Lookup(ctx context.Context, ref *corev1.RecordRef) (*corev1.RecordMeta, error) {
	logger.Debug("Looking up record in OCI store", "ref", ref)

	ociDigest, err := getDigestFromCID(ref.GetCid())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid record reference: %s", ref.GetCid())
	}

	// check if blob exists on remote
	{
		exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: ociDigest})
		if err != nil {
			if strings.Contains(err.Error(), "invalid reference") {
				return nil, status.Errorf(codes.InvalidArgument, "invalid record reference: %s", ref.GetCid())
			}

			return nil, status.Errorf(codes.Internal, "failed to check if record exists: %v", err)
		}

		logger.Debug("Checked if record exists in OCI store", "exists", exists)

		if !exists {
			return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
		}
	}

	// read manifest data from remote
	var manifest ocispec.Manifest
	{
		// resolve manifest from remote tag
		manifestDesc, err := s.repo.Resolve(ctx, ref.GetCid())
		if err != nil {
			logger.Error("Failed to resolve manifest", "error", err)

			// do not error here, as we may have a raw record stored but not tagged with
			// the manifest. only agents are tagged with manifests
			return nil, status.Errorf(codes.NotFound, "manifest not found: %s", ref.GetCid())
		}

		// TODO: validate manifest by size

		// fetch manifest from remote
		manifestRd, err := s.repo.Fetch(ctx, manifestDesc)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to fetch manifest: %v", err)
		}

		// read manifest
		manifestData, err := io.ReadAll(manifestRd)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to read manifest: %v", err)
		}

		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to unmarshal manifest: %v", err)
		}
	}

	// Extract record type from manifest metadata
	recordType, ok := manifest.Annotations[manifestDirObjectTypeKey]
	if !ok {
		return nil, status.Errorf(codes.Internal, "record type not found in manifest annotations: %s", manifestDirObjectTypeKey)
	}

	// Extract comprehensive metadata from manifest annotations using our enhanced parser
	recordMeta := parseManifestAnnotations(manifest.Annotations)

	// Set the CID from the request (this is the primary identifier)
	recordMeta.Cid = ref.GetCid()

	logger.Debug("Record metadata retrieved successfully", "cid", ref.GetCid(), "type", recordType)

	return recordMeta, nil
}

func (s *store) Pull(ctx context.Context, ref *corev1.RecordRef) (*corev1.Record, error) {
	logger.Debug("Pulling record from OCI store", "ref", ref)

	// First resolve the manifest using the CID as a tag (same as Lookup)
	manifestDesc, err := s.repo.Resolve(ctx, ref.GetCid())
	if err != nil {
		logger.Error("Failed to resolve manifest for pull", "cid", ref.GetCid(), "error", err)
		return nil, status.Errorf(codes.NotFound, "record not found: %s", ref.GetCid())
	}

	// Fetch the manifest to get the blob descriptor
	manifestReader, err := s.repo.Fetch(ctx, manifestDesc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch manifest: %v", err)
	}
	defer manifestReader.Close()

	// Read and parse the manifest
	manifestData, err := io.ReadAll(manifestReader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read manifest: %v", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal manifest: %v", err)
	}

	// Get the blob descriptor from the manifest layers
	if len(manifest.Layers) == 0 {
		return nil, status.Errorf(codes.Internal, "manifest has no layers")
	}

	// Use the first layer as the record blob
	blobDesc := manifest.Layers[0]

	// Fetch the record data using the correct blob descriptor from the manifest
	reader, err := s.repo.Fetch(ctx, blobDesc)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "record blob not found: %s", ref.GetCid())
	}
	defer reader.Close()

	// Read all data from the reader
	recordData, err := io.ReadAll(reader)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to read record data: %v", err)
	}

	// Unmarshal canonical JSON data back to Record
	record, err := corev1.CanonicalUnmarshal(recordData)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshal record: %v", err)
	}

	logger.Debug("Record pulled successfully", "cid", ref.GetCid(), "size", len(recordData))

	return record, nil
}

func (s *store) Delete(ctx context.Context, ref *corev1.RecordRef) error {
	logger.Debug("Deleting record from OCI store", "ref", ref)

	// Validate input
	if ref.GetCid() == "" {
		return status.Error(codes.InvalidArgument, "record CID is required")
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
