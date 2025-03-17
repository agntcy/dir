// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
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

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	"github.com/agntcy/dir/server/types"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
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

type store struct {
	repo oras.GraphTarget
}

func New(cfg ociconfig.Config) (types.StoreAPI, error) {
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
			}),
	}

	return &store{
		repo: repo,
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
func (s *store) Push(ctx context.Context, ref *coretypes.ObjectRef, contents io.Reader) (*coretypes.ObjectRef, error) {
	// push raw data
	blobRef, blobDesc, err := s.pushData(ctx, ref, contents)
	if err != nil {
		return nil, err
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
	_, err = oras.Tag(ctx, s.repo, manifestDesc.Digest.String(), refToCID(blobRef))
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
	// check if blob exists on remote
	{
		exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: ocidigest.Digest(ref.GetDigest())})
		if err != nil {
			return nil, err //nolint:wrapcheck
		}

		if !exists {
			return nil, types.ErrDigestNotFound
		}
	}

	// read manifest data from remote
	var manifest ocispec.Manifest
	{
		// resolve manifest from remote tag
		manifestDesc, err := s.repo.Resolve(ctx, refToCID(ref))
		if err != nil {
			// soft fail
			return ref, nil
		}

		// TODO: validate manifest by size

		// fetch manifest from remote
		manifestRd, err := s.repo.Fetch(ctx, manifestDesc)
		if err != nil {
			// soft fail
			return ref, nil
		}

		// read manifest
		manifestData, err := io.ReadAll(manifestRd)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			return nil, err
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
	return s.repo.Fetch(ctx, ocispec.Descriptor{ //nolint:wrapcheck
		Digest: ocidigest.Digest(ref.GetDigest()),
		Size:   int64(ref.GetSize()), //nolint:gosec
	})
}

func (s *store) Delete(_ context.Context, _ *coretypes.ObjectRef) error {
	// todo: remove tag (or remove tag with cleanup)
	// todo: remove manifest
	// todo: remove blob
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
	if err := s.repo.Push(ctx, blobDesc, rd); err != nil {
		return nil, ocispec.Descriptor{}, err
	}

	// return ref
	return &coretypes.ObjectRef{
		Digest: ref.GetDigest(),
		Type:   coretypes.ObjectType_OBJECT_TYPE_RAW.String(),
		Size:   uint64(blobDesc.Size),
	}, blobDesc, nil
}

// refToCID turns object digest into CID.
func refToCID(ref *coretypes.ObjectRef) string {
	hash, _ := mh.Sum([]byte(ref.GetDigest()), mh.SHA2_256, -1)
	c := cid.NewCidV1(cid.Raw, hash)

	return c.String()
}

// cleanMeta returns metadata without OCI- or Dir- annotations.
func cleanMeta(meta map[string]string) map[string]string {
	delete(meta, "org.opencontainers.image.created")
	delete(meta, manifestDirObjectTypeKey)
	// TODO: remove all keys that start with "manifestDirObjectKeyPrefix"

	return meta
}
