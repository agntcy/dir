// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"io"
	"net/http"

	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	ocidigest "github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

func stringPtr(s string) *string {
	return &s
}

// NewORASRepository creates a new ORAS repository client configured with authentication.
func NewORASRepository(cfg ociconfig.Config) (*remote.Repository, error) {
	repo, err := remote.NewRepository(fmt.Sprintf("%s/%s", cfg.RegistryAddress, cfg.RepositoryName))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote repo: %w", err)
	}

	// Configure repository
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

	return repo, nil
}

// pushOrSkip pushes data to the OCI registry if it does not already exist
func (s *store) pushOrSkip(ctx context.Context, reader io.ReadCloser, mediaType string) (ocidigest.Digest, error) {
	// Close reader when done
	defer reader.Close()

	// Read all data from the reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read data from reader: %w", err)
	}

	// Compute digest
	digest := ocidigest.FromBytes(data)

	// Check if data already exists
	exists, err := s.repo.Exists(ctx, ocispec.Descriptor{Digest: digest})
	if err != nil {
		return "", fmt.Errorf("failed to check blob existence: %w", err)
	}
	if !exists {
		_, err := oras.PushBytes(ctx, s.repo, mediaType, data)
		if err != nil {
			return "", fmt.Errorf("failed to push object bytes: %w", err)
		}
	}

	return digest, nil
}
