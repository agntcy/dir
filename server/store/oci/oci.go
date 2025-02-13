// SPDX-FileCopyrightText: Copyright (c) 2025 Cisco and/or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry/remote"

	coretypes "github.com/agntcy/dir/api/core/v1alpha1"
	"github.com/agntcy/dir/server/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type store struct {
	repository *remote.Repository
}

func New(config Config) (types.StoreService, error) {
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

func (s *store) Lookup(ctx context.Context, ref *coretypes.Digest) (*coretypes.ObjectMeta, error) {
	_, metaReader, err := s.repository.Blobs().FetchReference(ctx, string(ref.Value))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch object: %w", err)
	}

	// Read the blob
	metaBytes, err := io.ReadAll(metaReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	// Unmarshal the object
	meta := &coretypes.ObjectMeta{}
	if err := json.Unmarshal(metaBytes, meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object: %w", err)
	}

	return meta, nil
}

// TODO Currently, read contents is required to create an OCI descriptor for the object.
// Explore options to allow pushing objects without having to read the contents.
func (s *store) Push(ctx context.Context, meta *coretypes.ObjectMeta, contents io.Reader) (*coretypes.Digest, error) {
	// Read the contents to create an OCI descriptor
	contentsBytes, err := io.ReadAll(contents)
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to read contents: %w", err)
	}

	// Push contents to the repository
	desc := content.NewDescriptorFromBytes(ocispec.MediaTypeDescriptor, contentsBytes)
	err = s.repository.Push(ctx, desc, bytes.NewReader(contentsBytes))
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to push object: %w", err)
	}

	// Create metadata
	meta.Digest = &coretypes.Digest{
		Type:  coretypes.DigestType_DIGEST_TYPE_SHA256,
		Value: []byte(desc.Digest.String()),
	}
	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Push metadata to the repository
	metaDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeDescriptor, metaBytes)
	err = s.repository.Push(ctx, metaDesc, bytes.NewReader(metaBytes))
	if err != nil {
		return &coretypes.Digest{}, fmt.Errorf("failed to push object: %w", err)
	}

	return &coretypes.Digest{
		Type:  coretypes.DigestType_DIGEST_TYPE_SHA256,
		Value: []byte(metaDesc.Digest.String()),
	}, nil
}

func (s *store) Pull(ctx context.Context, ref *coretypes.Digest) (io.Reader, error) {
	meta, err := s.Lookup(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup object: %w", err)
	}

	_, reader, err := s.repository.Blobs().FetchReference(ctx, string(meta.Digest.Value))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch object: %w", err)
	}

	return reader, nil
}

func (s *store) Delete(ctx context.Context, ref *coretypes.Digest) error {
	meta, err := s.Lookup(ctx, ref)
	if err != nil {
		return fmt.Errorf("failed to lookup object: %w", err)
	}

	// Delete the metadata
	metaDescriptor, err := s.repository.Blobs().Resolve(ctx, string(ref.Value))
	if err != nil {
		return fmt.Errorf("failed to resolve object: %w", err)
	}
	if err := s.repository.Delete(ctx, metaDescriptor); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	// Delete the blob
	objectDescriptor, err := s.repository.Blobs().Resolve(ctx, string(meta.Digest.Value))
	if err != nil {
		return fmt.Errorf("failed to resolve object: %w", err)
	}
	if err := s.repository.Delete(ctx, objectDescriptor); err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
