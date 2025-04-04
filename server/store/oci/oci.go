// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

//nolint:wrapcheck,nilerr,gosec
package oci

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/datastore"
	"github.com/agntcy/dir/server/store/cache"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/agntcy/dir/utils/logging"
	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

const (
	// Used for dir-specific annotations.
	manifestDirObjectKeyPrefix = "org.agntcy.dir"
	manifestDirObjectTypeKey   = manifestDirObjectKeyPrefix + "/type"
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

// Push object to the OCI registry
//
// This creates a blob, a manifest that points to that blob, and a tagged release for that manifest.
// The tag for the manifest is: <CID of digest>.
// The tag for the blob is needed to link the actual object with its associated metadata.
// Note that metadata can be stored in a different store and only wrap this store.
//
// Ref: https://github.com/oras-project/oras-go/blob/main/docs/Modeling-Artifacts.md
func (s *store) Push(ctx context.Context, ref *coretypes.ObjectRef, contents io.Reader) (*coretypes.ObjectRef, error) {
	logger.Debug("Pushing object to OCI store", "ref", ref)

	// push raw data
	blobRef, blobDesc, err := s.pushData(ctx, ref, contents)
	if err != nil {
		return nil, fmt.Errorf("failed to push data: %w", err)
	}

	// set annotations for manifest
	annotations := cleanMeta(ref.GetAnnotations())
	annotations[manifestDirObjectTypeKey] = ref.GetType()

	// push manifest
	manifestDesc, err := oras.PackManifest(ctx, s.repo, oras.PackManifestVersion1_1, ocispec.MediaTypeImageManifest,
		oras.PackManifestOptions{
			ManifestAnnotations: annotations,
			Layers: []ocispec.Descriptor{
				blobDesc,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	// tag manifest
	// tag => resolves manifest to object which can be looked up (lookup)
	// tag => allows to pull object directly (pull)
	// tag => allows listing and filtering tags (list)
	_, err = oras.Tag(ctx, s.repo, manifestDesc.Digest.String(), ref.GetShortRef())
	if err != nil {
		return nil, err
	}

	// return clean ref
	return &coretypes.ObjectRef{
		Digest:      blobRef.GetDigest(),
		Type:        ref.GetType(),
		Size:        ref.GetSize(),
		Annotations: cleanMeta(ref.GetAnnotations()),
	}, nil
}

// Lookup checks if the ref exists as a tagged object.
func (s *store) Lookup(ctx context.Context, ref *coretypes.ObjectRef) (*coretypes.ObjectRef, error) {
	logger.Debug("Looking up object in OCI store", "ref", ref)

	// check if blob exists on remote
	{
		exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: ocidigest.Digest(ref.GetDigest())})
		if err != nil {
			return nil, fmt.Errorf("failed to check if object exists: %w", err)
		}

		logger.Debug("Checked if object exists in OCI store", "exists", exists)

		if !exists {
			return nil, types.ErrDigestNotFound
		}
	}

	// read manifest data from remote
	var manifest ocispec.Manifest
	{
		shortRef := ref.GetShortRef()

		// resolve manifest from remote tag
		manifestDesc, err := s.repo.Resolve(ctx, shortRef)
		if err != nil {
			logger.Error("Failed to resolve manifest", "error", err)

			// do not error here, as we may have a raw object stored but not tagged with
			// the manifest. only agents are tagged with manifests
			return ref, nil
		}

		// TODO: validate manifest by size

		// fetch manifest from remote
		manifestRd, err := s.repo.Fetch(ctx, manifestDesc)
		if err != nil {
			return ref, fmt.Errorf("failed to fetch manifest: %w", err)
		}

		// read manifest
		manifestData, err := io.ReadAll(manifestRd)
		if err != nil {
			return nil, fmt.Errorf("failed to read manifest: %w", err)
		}

		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
		}
	}

	// read object size from manifest
	var objectSize uint64
	if len(manifest.Layers) > 0 {
		objectSize = uint64(manifest.Layers[0].Size) //nolint:gosec
	}

	// read object type from manifest metadata
	objectType, ok := manifest.Annotations[manifestDirObjectTypeKey]
	if !ok {
		return nil, errors.New("not a dir-specific object")
	}

	// return clean ref
	return &coretypes.ObjectRef{
		Digest:      ref.GetDigest(),
		Type:        objectType,
		Size:        objectSize,
		Annotations: cleanMeta(manifest.Annotations),
	}, nil
}

func (s *store) Pull(ctx context.Context, ref *coretypes.ObjectRef) (io.ReadCloser, error) {
	logger.Debug("Pulling object from OCI store", "ref", ref)

	return s.repo.Fetch(ctx, ocispec.Descriptor{ //nolint:wrapcheck
		Digest: ocidigest.Digest(ref.GetDigest()),
		Size:   int64(ref.GetSize()), //nolint:gosec
	})
}

func (s *store) Delete(ctx context.Context, ref *coretypes.ObjectRef) error {
	logger.Debug("Deleting object from OCI store", "ref", ref)

	switch repo := s.repo.(type) {
	case *oci.Store:
		return s.deleteFromOCIStore(ctx, repo, ref)
	case *remote.Repository:
		return s.deleteFromRemoteRepository(ctx, repo, ref)
	default:
		return fmt.Errorf("unsupported repo type: %T", s.repo)
	}
}

// deleteFromOCIStore handles deletion of objects from an OCI store.
func (s *store) deleteFromOCIStore(ctx context.Context, store *oci.Store, ref *coretypes.ObjectRef) error {
	// Untag the manifest. Errors are logged but not returned because
	// the object may exist without being tagged with a manifest.
	if err := store.Untag(ctx, ref.GetShortRef()); err != nil {
		logger.Debug("Failed to untag manifest", "error", err)
	}

	// Resolve and delete the manifest. Errors are logged but not returned
	// for the same reason as above.
	manifestDesc, err := s.repo.Resolve(ctx, ref.GetShortRef())
	if err != nil {
		logger.Debug("Failed to resolve manifest", "error", err)
	} else if err := store.Delete(ctx, manifestDesc); err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}

	// Delete the blob associated with the descriptor.
	blobDesc := ocispec.Descriptor{
		Digest: ocidigest.Digest(ref.GetDigest()),
	}
	if err := store.Delete(ctx, blobDesc); err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	return nil
}

// deleteFromRemoteRepo handles deletion of objects from a remote repository.
func (s *store) deleteFromRemoteRepository(ctx context.Context, repo *remote.Repository, ref *coretypes.ObjectRef) error {
	// Resolve and delete the manifest. Errors are logged but not returned because
	// the object may exist without being tagged with a manifest.
	manifestDesc, err := s.repo.Resolve(ctx, ref.GetShortRef())
	if err != nil {
		logger.Debug("Failed to resolve manifest", "error", err)
	} else if err := repo.Manifests().Delete(ctx, manifestDesc); err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}

	// Delete the blob associated with the descriptor.
	blobDesc := ocispec.Descriptor{
		Digest: ocidigest.Digest(ref.GetDigest()),
	}
	if err := repo.Blobs().Delete(ctx, blobDesc); err != nil {
		return fmt.Errorf("failed to delete blob: %w", err)
	}

	return nil
}

// pushData pushes raw data to OCI.
func (s *store) pushData(ctx context.Context, ref *coretypes.ObjectRef, rd io.Reader) (*coretypes.ObjectRef, ocispec.Descriptor, error) {
	// push blob
	blobDesc := ocispec.Descriptor{
		MediaType: "application/octet-stream",
		Digest:    ocidigest.Digest(ref.GetDigest()),
		Size:      int64(ref.GetSize()),
	}

	logger.Debug("Pushing blob to OCI store", "ref", ref, "blobDesc", blobDesc)

	err := s.repo.Push(ctx, blobDesc, rd)
	if err != nil {
		logger.Error("Failed to push blob to OCI store", "error", err)

		// return only for non-valid errors
		if !strings.Contains(err.Error(), "already exists") {
			return nil, ocispec.Descriptor{}, err
		}
	}

	// return ref
	return &coretypes.ObjectRef{
		Digest: ref.GetDigest(),
		Type:   ref.GetType(),
		Size:   uint64(blobDesc.Size),
	}, blobDesc, nil
}

// cleanMeta returns metadata without OCI- or Dir- annotations.
func cleanMeta(meta map[string]string) map[string]string {
	if meta == nil {
		return map[string]string{}
	}

	// delete all OCI-specific metadata
	delete(meta, "org.opencontainers.image.created")

	// delete all Dir-specific metadata
	delete(meta, manifestDirObjectTypeKey)
	// TODO: clean all with dir prefix

	return meta
}
