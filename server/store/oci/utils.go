// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package oci

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	corev1 "github.com/agntcy/dir/api/core/v1"
	ociconfig "github.com/agntcy/dir/server/store/oci/config"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/sync/errgroup"
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

func (s *store) descLookupMany(ctx context.Context, cids []*corev1.RecordRef) (map[string]ocispec.Descriptor, error) {
	results := make(map[string]ocispec.Descriptor)
	resultsMu := sync.Mutex{}

	// Use errgroup for concurrent fetching with controlled parallelism
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(10) // Limit concurrent requests

	for _, cid := range cids {
		cid := cid // Capture loop variable
		g.Go(func() error {
			// Fetch content
			descriptor, err := s.repo.Resolve(ctx, cid.GetCid())
			if err != nil {
				return fmt.Errorf("failed to fetch %s: %w", cid, err)
			}

			// Store result
			resultsMu.Lock()
			results[cid.GetCid()] = descriptor
			resultsMu.Unlock()

			return nil
		})
	}

	// Wait for all fetches to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

func (s *store) safePushByte(ctx context.Context, desc ocispec.Descriptor, content []byte) error {
	_, err := s.repo.Push(ctx, desc, content)
	if err != nil {
		return fmt.Errorf("failed to push content: %w", err)
	}
	return nil
}
