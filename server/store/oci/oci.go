// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"context"
	"fmt"
	storetypes "github.com/agntcy/dir/api/store/v1alpha1"
	"io"

	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"

	"github.com/agntcy/dir/server/types"
	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type store struct {
	repository *remote.Repository
}

func New(config Config) (types.StoreAPI, error) {
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", config.RegistryAddress, config.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// TODO: Make configurable
	repo.PlainHTTP = true

	// TODO Set the client not to use the default client
	// repo.Client = &auth.Client{
	// 	Client: retry.DefaultClient,
	// 	Header: http.Header{
	// 		"User-Agent": {"oras-go"},
	// 	},
	// 	Cache: auth.DefaultCache,
	// 	Credential: auth.StaticCredential(
	// 		"",
	// 		auth.Credential{
	// 			Username: config.Zot.Username,
	// 			Password: config.Zot.Password,
	// 		}),
	// }

	return &store{
		repository: repo,
	}, nil
}

func (s *store) Lookup(ctx context.Context, ref *storetypes.ObjectRef) (*storetypes.ObjectRef, error) {
	digest, err := convertToOciDigest(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to convert digest: %w", err)
	}

	descriptor, err := s.repository.Blobs().Resolve(ctx, digest.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch object: %w", err)
	}

	return convertToRef(descriptor), nil
}

// Push object to the OCI.
//
// TODO: Currently, full read is required to create an OCI descriptor.
func (s *store) Push(ctx context.Context, ref *storetypes.ObjectRef, contents io.Reader) (*storetypes.ObjectRef, error) {
	// Read the contents to create an OCI descriptor
	contentsBytes, err := io.ReadAll(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to read contents: %w", err)
	}

	// Push contents to the repository
	desc := content.NewDescriptorFromBytes(ocispec.MediaTypeDescriptor, contentsBytes)
	desc.Annotations = map[string]string{
		"name": *ref.Name,
		"type": *ref.Type,
	}
	err = s.repository.Push(ctx, desc, bytes.NewReader(contentsBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to push object: %w", err)
	}

	return convertToRef(desc), nil
}

func (s *store) Pull(ctx context.Context, ref *storetypes.ObjectRef) (io.Reader, error) {
	digest, err := convertToOciDigest(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to convert digest: %w", err)
	}

	_, reader, err := s.repository.Blobs().FetchReference(ctx, digest.String())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch object: %w", err)
	}

	return reader, nil
}

func (s *store) Delete(ctx context.Context, ref *storetypes.ObjectRef) error {
	digest, err := convertToOciDigest(ref)
	if err != nil {
		return fmt.Errorf("failed to convert digest: %w", err)
	}

	// Delete the blob
	objectDescriptor, err := s.repository.Blobs().Resolve(ctx, digest.Encoded())
	if err != nil {
		return fmt.Errorf("failed to resolve object: %w", err)
	}
	if err := s.repository.Delete(ctx, objectDescriptor); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// convertToOciDigest converts an ObjectRef to an ocidigest.Digest
func convertToOciDigest(ref *storetypes.ObjectRef) (*ocidigest.Digest, error) {
	digest := ocidigest.Digest(ref.Digest)
	if err := digest.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate OCI digest: %w", err)
	}
	return &digest, nil
}

func convertToRef(descriptor ocispec.Descriptor) *storetypes.ObjectRef {
	size := uint64(descriptor.Size)
	return &storetypes.ObjectRef{
		Digest: descriptor.Digest.String(),
		Name:   ptrTo[string](descriptor.Annotations["name"]),
		Type:   ptrTo[string](descriptor.Annotations["type"]),
		Size:   &size,
	}
}

func ptrTo[T any](value T) *T {
	return &value
}
